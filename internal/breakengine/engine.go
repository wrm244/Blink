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
	"blink/internal/stats"
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
	// autoPaused 标记当前暂停是否由会议/媒体检测自动触发。
	// 为 true 时条件消失会自动恢复；用户手动 Pause 时它为 false。
	autoPaused bool
	// meeting/media 记录最近一次外部检测的结果，用于边沿检测
	// （条件从不活跃变为活跃时触发自动暂停）。
	meeting bool
	media   bool
	// pausedRemaining 让 Pause/Resume 在不依赖时钟运行的情况下冻结倒计时。
	pausedRemaining time.Duration
	// pausedAccum 累计当前阶段内处于暂停的总时长，用于从统计的
	// 实际专注时长中扣除暂停部分。setPhase 时清零。
	pausedAccum time.Duration
	// pausedAt 记录本次暂停开始的时刻，Resume 时累加到 pausedAccum。
	pausedAt time.Time

	// lastTick 用于检测系统休眠（突然出现的大间隔）。
	lastTick time.Time

	// statsStore 记录每日专注/休息时长，供前端日历统计读取。
	// 可能为 nil（未启用统计时），所有记录方法对 nil 安全。
	statsStore *stats.Store
	// phaseStart 记录当前阶段的开始时刻，用于在阶段结束时
	// 计算实际持续时长并累加到统计。setPhase 每次切换都会刷新它。
	phaseStart time.Time

	// app 是 Wails 应用实例，在 Start 时取得；遮罩/通知窗口的句柄
	// 不放在这里——它们由 windowLoop 独占（见 windows.go 的 windowState）。
	app *application.App

	// audioProbe 是会议/媒体检测的探针实现。以字段而非直接调用
	// platform.AudioActivity 的形式存在，是为了让单测能替换掉真正的
	// CGo 探针（那会去遍历系统音频进程），从而断言"什么时候该查、
	// 什么时候不该查"而无需触碰系统状态。
	audioProbe func() (meeting, media bool)

	// lastEmitted 是上一次真正投递出去的状态快照，emit 据此跳过
	// 内容完全相同的重复事件。
	lastEmitted State
	// emitCount 累计真正投递的 tick 事件次数。emit 的去重逻辑必须
	// 可观测，否则"少发事件"和"发错事件"在测试里长得一样；
	// 生产代码不读取它。
	emitCount int

	stopCh  chan struct{}
	cmdCh   chan windowCmd
	started bool
}

// New 根据给定设置创建引擎。
func New(s config.Settings) *Engine {
	return &Engine{settings: s, audioProbe: platform.AudioActivity}
}

// SetStatsStore 注入统计存储。必须在 Start 之前调用。
// 传入 nil 则禁用统计记录（记录方法对 nil 安全）。
func (e *Engine) SetStatsStore(store *stats.Store) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.statsStore = store
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
	// stopCh / cmdCh 属于"这一代"后台循环。必须以参数形式交给下面
	// 启动的 goroutine，而不是让它们回头去读 e.stopCh / e.cmdCh 字段：
	// Stop() 关闭的只是当前这一代的 channel，而 Start() 会立刻换上新的。
	// 若 goroutine 读的是字段，那么 Stop→Start 之后回到 select 的旧循环
	// 读到的会是新一代（未关闭）的 channel，于是永远等下去——同一个引擎
	// 从此有两套循环在驱动状态机（tick 翻倍、探测翻倍、emit 翻倍），
	// 且因为"停止后再开始"通常是最后一次 Start，僵尸循环会活到进程退出。
	stopCh := make(chan struct{})
	cmdCh := make(chan windowCmd, 8)
	e.stopCh = stopCh
	e.cmdCh = cmdCh
	e.app = application.Get()
	now := time.Now()
	e.lastTick = now
	// 清零去重缓存：引擎重启后的第一次 emit 必须真正投递出去，
	// 否则若新状态恰好等于停止前那次快照，前端就收不到这次重启。
	e.lastEmitted = State{}
	e.resetExternalState()
	e.startFocus()
	e.mu.Unlock()

	go e.loop(stopCh)
	go e.idleLoop(stopCh)
	go e.windowLoop(stopCh, cmdCh)
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
	st := State{Phase: e.phase, Idle: e.idle, Paused: e.paused, AutoPaused: e.autoPaused, Meeting: e.meeting, MediaPlaying: e.media, ShortBreakCount: e.breaksDone}
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
//
// 内容与上次完全相同时跳过投递。暂停、空闲这类冻结状态下每秒算出的
// 快照一模一样，逐秒投递只是让 Go 侧多做一次 JSON 序列化、让事件总线
// 多广播一轮、让每个 webview 多赋一组相同的值。所有下游都不依赖这些
// 重复事件：托盘自带变更检测，提醒窗口走本地时钟，遮罩与设置页把值
// 直接交给 Vue 的响应式系统（值未变即不重渲染）。
func (e *Engine) emit() {
	st := e.state()
	if st == e.lastEmitted {
		return
	}
	e.lastEmitted = st
	e.emitCount++
	a := e.app_()
	if a == nil {
		return
	}
	a.Event.Emit(eventTick, st)
}

// loop 是每秒驱动一次的主循环，同时承担会议/媒体检测。
// stopCh 由 Start 传入并只对应本代循环，详见 Start 中的说明。
//
// 外部检测原本由独立的 externalSuspendLoop（同为 1 秒周期）承担，
// 合并进来有三个好处：少一个后台 goroutine；每秒只获取一次引擎锁
// （原本两个循环各抢一次，且相位随机）；以及确定执行次序——先检测
// 再推进，本轮检测出的自动暂停能立刻被这次 tick 看到，而不必等到
// 另一个 goroutine 的下一次唤醒。
func (e *Engine) loop(stopCh chan struct{}) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case now := <-ticker.C:
			e.tickOnce(now)
		}
	}
}

// tickOnce 执行一个完整周期：会议/媒体检测 → 状态机推进（tick 内部会
// 投递事件）。
//
// 单独抽成方法而不是内联在 loop 里，是为了让"探测期间不得持有引擎锁"
// 这类并发契约能被测试直接验证——否则测试只能去驱动真实的 ticker。
func (e *Engine) tickOnce(now time.Time) {
	// 探测在锁外完成（CGo 调用约 2ms，见 probeExternal 的说明），
	// 只把结果的应用与状态机推进留在临界区内。
	meeting, media, probed := e.probeExternal()
	e.mu.Lock()
	e.applyExternalResult(meeting, media, probed)
	e.tick(now)
	e.mu.Unlock()
}

// idleLoop 定期采样系统空闲时间，当用户不活动超过阈值时
// 标记引擎为空闲状态（保持/重置专注计时器）。活动恢复时
// 清除标记并开始新的专注周期。
func (e *Engine) idleLoop(stopCh chan struct{}) {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
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
					// 进入空闲前，记录已专注的时长（否则这部分会丢失）。
					if e.phase == PhaseFocusing && !e.phaseStart.IsZero() {
						e.recordFocus()
					}
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
