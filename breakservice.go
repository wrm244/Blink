package main

import (
	"context"
	"log"
	"os/exec"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
	"blink/internal/config"
	"blink/internal/platform"
	"blink/internal/stats"
)

// BreakService 是前端通过 Wails 绑定调用的 Go 类型。
// 它是 break engine 之上的薄封装：engine 拥有计时状态机，
// 此服务仅转发用户操作和设置。
type BreakService struct {
	engine *breakengine.Engine
}

// NewBreakService 创建一个新的 BreakService。
func NewBreakService(engine *breakengine.Engine) *BreakService {
	return &BreakService{engine: engine}
}

// ServiceName 供 Wails 日志使用。
func (s *BreakService) ServiceName() string { return "BreakService" }

// ServiceStartup 由 Wails 在应用启动后调用。
//
// 启动时始终打开设置窗口——不默认驻留托盘，让用户能直接看到并操作界面。
// 引擎是否同时开始倒计时仍取决于引导状态与 AutoStart：已完成引导且开启
// 自动开始时启动，否则等待 CompleteOnboarding()。
func (s *BreakService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	settings := s.engine.GetSettings()
	showPreferences()
	if settings.Onboarded && settings.AutoStart {
		s.engine.Start()
	}
	return nil
}

// GetState 返回引擎的当前快照供 UI 渲染。
func (s *BreakService) GetState() breakengine.State { return s.engine.GetState() }

// GetSettings 返回当前设置。
func (s *BreakService) GetSettings() config.Settings { return s.engine.GetSettings() }

// SaveSettings 持久化设置，应用到运行中的引擎并重新绑定全局快捷键。
// 快捷键重绑在自己的 goroutine 上运行，因为绑定会调度到主线程，
// 而此方法本身就在主线程上运行（它是 Wails 绑定调用）。
func (s *BreakService) SaveSettings(settings config.Settings) error {
	if err := config.Save(settings); err != nil {
		return err
	}
	s.engine.ApplySettings(settings)
	go func() {
		registerAll(settings)
		// 语言为空表示"跟随系统"：resolveMenuLanguage 内部按系统语言解析，
		// 因此无条件同步托盘菜单语言（否则用户把语言改回跟随系统后，
		// 托盘仍停留在旧语言）。
		setMenuLanguage(settings.Language)
		// 开机自启属于磁盘级系统操作（登录项/注册表），放到后台同步，
		// 避免阻塞主线程上的保存流程。dev 模式（裸二进制）下
		// SetLaunchAtLogin 会静默失败，不中断正常保存。
		if err := platform.SetLaunchAtLogin(settings.LaunchAtLogin); err != nil {
			log.Printf("blink: 更新开机自启失败：%v", err)
		}
	}()
	return nil
}

// CompleteOnboarding 标记引导完成并首次启动引擎。
// 在用户完成首次设置界面后调用一次。
func (s *BreakService) CompleteOnboarding() error {
	settings := s.engine.GetSettings()
	settings.Onboarded = true
	if err := config.Save(settings); err != nil {
		return err
	}
	s.engine.ApplySettings(settings)
	go registerAll(settings)
	go setMenuLanguage(settings.Language)
	s.engine.Start()
	return nil
}

// StartEngine 启动倒计时（手动停止或暂停后使用）。
func (s *BreakService) StartEngine() { s.engine.Start() }

// StopEngine 完全停止倒计时。
func (s *BreakService) StopEngine() { s.engine.Stop() }

// StartBreakNow 立即开始休息，跳过休息前提醒。
func (s *BreakService) StartBreakNow() { s.engine.StartBreakNow() }

// SkipBreak 结束当前休息或休息前提醒。
func (s *BreakService) SkipBreak() { s.engine.SkipBreak() }

// PostponeBreak 取消当前/即将到来的休息并重新开始专注周期。
func (s *BreakService) PostponeBreak() { s.engine.PostponeBreak() }

// Pause 冻结倒计时。
func (s *BreakService) Pause() { s.engine.Pause() }

// Resume 继续暂停的倒计时。
func (s *BreakService) Resume() { s.engine.Resume() }

// Reset 清除长休息计数器并重新开始专注周期。
func (s *BreakService) Reset() { s.engine.Reset() }

// AckClose 由前端在收到 blink:close-request、确认弹窗已显示后立即调用，
// 用来停掉 Go 端的兜底定时器。
//
// 拆成"接管 + 决定"两步，是为了让兜底超时只衡量前端的响应能力，
// 而不把用户对着弹窗思考的时间也算进去（详见 prefs_close.go）。
func (s *BreakService) AckClose() { ackCloseAsk() }

// ResolveClose 接收前端关闭确认弹窗的结果，决定设置窗口关闭后的去向。
//
// action 取 config.CloseActionBackground（保留在后台运行）、
// config.CloseActionQuit（完全退出）或 closeActionCancel（用户取消，
// 窗口保持打开）。remember 为 true 时把这次选择写入设置，之后关闭窗口
// 不再询问——用户可在"选项 → 关闭窗口时"改回每次询问。
//
// 该流程目前只有 Windows 会触发（其它平台不安装关闭钩子，前端也就
// 收不到 blink:close-request 事件），但方法本身是平台无关的。
//
// 线程：本方法是 Wails 绑定调用，运行在主线程上。窗口 Hide 与 app.Quit
// 内部都会 InvokeSync 回主线程，直接调用会死锁，因此实际动作必须甩到
// goroutine 上执行（与 ShowWindow 同理）。
func (s *BreakService) ResolveClose(action string, remember bool) error {
	// 认领这次询问。若兜底逻辑已经先一步处理（前端回应超时），
	// 这里就不能再动手，否则会和兜底重复执行。
	if !finishCloseAsk() {
		return nil
	}
	// 取消，以及任何无法识别的动作，都保持窗口原样——
	// 对一个"关不掉窗口"的误操作来说，什么都不做是最安全的降级。
	if action != config.CloseActionBackground && action != config.CloseActionQuit {
		return nil
	}
	if remember {
		settings := s.engine.GetSettings()
		settings.CloseAction = action
		if err := config.Save(settings); err != nil {
			return err
		}
		s.engine.ApplySettings(settings)
	}
	go func() {
		if action == config.CloseActionQuit {
			quitApp()
			return
		}
		hidePrefsWindow()
	}()
	return nil
}

// ShowWindow 显示设置窗口。由前端在数据加载完成、界面渲染就绪后调用，
// 避免窗口先白屏再显示内容。Show/Focus 在独立 goroutine 上执行，
// 因为它们内部通过 InvokeSync 调度到主线程，而本绑定方法本身就在
// 主线程执行，直接调用会死锁。
func (s *BreakService) ShowWindow() {
	go func() {
		if w, ok := app.Window.GetByName(prefsWindowName); ok && w != nil {
			w.Show()
			w.Focus()
		}
	}()
}

// OpenURL 用系统默认浏览器打开外部链接（如 GitHub 仓库页）。
// WebView 内的 window.open 在 WKWebView 中不会调起系统浏览器，
// 因此外部链接必须经由此方法交给操作系统处理。
func (s *BreakService) OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// GetMonthlyStats 返回指定年月每天的专注/休息统计。
// year 为完整年份（如 2026），month 为 1-12。
// 前端据此渲染日历热力图。返回的 map key 为日期号（1-31）。
func (s *BreakService) GetMonthlyStats(year int, month int) map[int]stats.DayStats {
	return s.engine.GetMonthlyStats(year, month)
}

// GetDayStats 返回指定日期的统计。year 为完整年份，month 为 1-12，day 为 1-31。
func (s *BreakService) GetDayStats(year int, month int, day int) stats.DayStats {
	return s.engine.GetDayStats(year, month, day)
}

// GetPendingNav 返回并清除待处理的目标 tab 名。
// 由托盘菜单的"统计"项设置，前端在 ready 后调用一次以切换到对应标签页。
// 返回空串表示无待处理导航。
func (s *BreakService) GetPendingNav() string {
	return takePendingNav()
}
