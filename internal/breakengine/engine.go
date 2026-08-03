// Package breakengine 包含 Blink 的核心计时状态机：
// 在专注（focus）→ 休息前提醒（pre-break）→ 短休息/长休息之间循环，
// 用户空闲时自动暂停，系统休眠后恢复，并驱动全屏休息遮罩窗口。
package breakengine

import (
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/config"
	"blink/internal/platform"
)

const (
	eventTick = "blink:tick"

	// 内置 macOS 系统音效：休息结束时播放柔和提示音，
	// 休息前提醒出现时播放轻柔的滴答声。
	soundBreakEnd = "Glass"
	soundPreBreak = "Tink"

	tickInterval      = 1 * time.Second  // 每秒滴答一次
	idleCheckInterval = 5 * time.Second   // 每 5 秒检查一次空闲状态
	// sleepGap：如果两次 tick 之间间隔超过此值，说明机器可能休眠了；
	// 与其触发一个过期的休息，不如重置专注周期。
	sleepGap = 5 * time.Second
)

// Engine 是休息提醒的状态机。它是并发安全的：
// 每个导出方法都获取互斥锁，后台 ticker 是唯一的其它写入者。
type Engine struct {
	mu       sync.Mutex
	settings config.Settings

	phase    Phase
	phaseEnd time.Time // 当前阶段的绝对结束时间
	total    time.Duration

	// breaksDone 记录自上次长休息以来完成的短休息次数。
	breaksDone int

	idle   bool
	paused bool
	// pausedRemaining 让 Pause/Resume 在不依赖时钟运行的情况下冻结倒计时。
	pausedRemaining time.Duration

	// lastTick 用于检测系统休眠（突然出现的大间隔）。
	lastTick time.Time

	app      *application.App
	overlays []application.Window
	notice   application.Window

	stopCh  chan struct{}
	cmdCh   chan windowCmd
	started bool
}

// New 根据给定设置创建引擎。
func New(s config.Settings) *Engine {
	return &Engine{settings: s}
}

// Start 启动后台 ticker 和空闲检测器。该方法是幂等的，
// 因此从服务的 ServiceStartup 钩子中调用是安全的。
func (e *Engine) Start() {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return
	}
	e.started = true
	e.stopCh = make(chan struct{})
	e.cmdCh = make(chan windowCmd, 8)
	e.app = application.Get()
	now := time.Now()
	e.lastTick = now
	e.startFocus()
	e.mu.Unlock()

	go e.loop()
	go e.idleLoop()
	go e.windowLoop()
}

// IsStarted 报告引擎的后台 ticker 是否正在运行。
func (e *Engine) IsStarted() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.started
}

// Stop 停止后台 goroutine。可以通过 Start 重新启动。
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return
	}
	e.started = false
	// 在关闭 stopCh 之前先入队隐藏命令，这样 windowLoop 有机会
	// 在退出前处理它们（参见 windowLoop），否则 windowLoop 可能在
	// 隐藏命令执行前就返回，导致遮罩窗口残留在屏幕上。
	e.hideNotice()
	e.hideOverlays()
	close(e.stopCh)
	e.mu.Unlock()
}

// app_ 返回缓存的 Wails 应用实例，在 Start 时设置。
func (e *Engine) app_() *application.App {
	return e.app
}

// ---- 状态快照 ----

// state 生成当前引擎状态的快照。调用者需持有互斥锁。
func (e *Engine) state() State {
	st := State{Phase: e.phase, Idle: e.idle, Paused: e.paused, ShortBreakCount: e.breaksDone}
	if e.settings.EnableLongBreaks {
		st.BreaksUntilLong = e.settings.LongBreakInterval - e.breaksDone
		if st.BreaksUntilLong < 0 {
			st.BreaksUntilLong = 0
		}
	}
	total := e.total
	remaining := total
	// 空闲和暂停阶段冻结倒计时：跳过 phaseEnd 查询，
	// 让 remaining 保持在 total。
	if !e.phaseEnd.IsZero() && e.phase != PhaseIdle {
		remaining = time.Until(e.phaseEnd)
	}
	if e.paused {
		remaining = e.pausedRemaining
	}
	if remaining < 0 {
		remaining = 0
	}
	st.TotalSec = int(total / time.Second)
	st.RemainingSec = int(remaining / time.Second)
	return st
}

// emit 向前端发送 tick 事件。调用者需持有互斥锁。
func (e *Engine) emit() {
	a := e.app_()
	if a == nil {
		return
	}
	a.Event.Emit(eventTick, e.state())
}

// loop 是每秒驱动一次的主循环。
func (e *Engine) loop() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case now := <-ticker.C:
			e.mu.Lock()
			e.tick(now)
			e.mu.Unlock()
		}
	}
}

// idleLoop 定期采样系统空闲时间，当用户不活动超过阈值时
// 标记引擎为空闲状态（保持/重置专注计时器）。活动恢复时
// 清除标记并开始新的专注周期。
func (e *Engine) idleLoop() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			idle := time.Duration(platform.IdleSeconds()) * time.Second
			e.mu.Lock()
			threshold := time.Duration(e.settings.IdleThresholdMin) * time.Minute
			wasIdle := e.idle
			if idle >= threshold {
				e.idle = true
				// 仅对专注流程执行空闲暂停；不打断正在进行的休息，
				// 也不影响用户手动暂停。检查 e.paused（实时布尔值）
				// 而非阶段值，因为 Pause() 在不改变阶段的情况下冻结倒计时，
				// 所以暂停的引擎在这里仍然报告 PhaseFocusing。
				if e.phase != PhaseIdle && !e.phase.IsBreak() && !e.paused {
					e.phase = PhaseIdle
					log.Printf("blink/breakengine: 空闲阈值达到（%s），专注计时器保持满值", threshold)
					e.hideNotice()
					e.hideOverlays()
					e.emit()
				}
			} else {
				e.idle = false
				if wasIdle && e.phase == PhaseIdle {
					log.Printf("blink/breakengine: 活动恢复，重新开始专注周期")
					e.startFocus()
				}
			}
			e.mu.Unlock()
		}
	}
}
