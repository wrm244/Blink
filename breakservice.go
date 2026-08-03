package main

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
	"blink/internal/config"
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

// ServiceStartup 由 Wails 在应用启动后调用。仅当用户已完成引导
//（且 AutoStart 开启）时引擎才开始倒计时；否则等待 CompleteOnboarding()。
func (s *BreakService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	settings := s.engine.GetSettings()
	if settings.Onboarded && settings.AutoStart {
		s.engine.Start()
	} else if !settings.Onboarded {
		// 首次运行：显示设置窗口，让用户在倒计时开始前完成配置。
		showPreferences()
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
		if settings.Language != "" {
			setMenuLanguage(settings.Language)
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
	if settings.Language != "" {
		go setMenuLanguage(settings.Language)
	}
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
