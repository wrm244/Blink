package breakengine

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// overlayOptions 构建全屏休息遮罩窗口的配置项。
// 使用不透明的深色背景（非半透明），以避免 macOS Tahoe 在窗口边缘
// 产生的明亮玻璃边框。窗口层级设为 ScreenSaver（最高），确保覆盖
// Dock 和菜单栏。
func overlayOptions(s *application.Screen) application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:           overlayName(s.ID),
		Title:          "",
		Frameless:      true,
		AlwaysOnTop:    true,
		DisableResize:  true,
		Hidden:         true,
		URL:            "/#break",
		Width:          s.Bounds.Width,
		Height:         s.Bounds.Height,
		X:              s.Bounds.X,
		Y:              s.Bounds.Y,
		InitialPosition: application.WindowXY,
		// 实色深色背景：与 CSS 渐变匹配，避免 webview 绘制前的颜色闪烁。
		// 不透明（非半透明）以避免 macOS Tahoe 在窗口边缘的明亮玻璃边框。
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour:  application.NewRGB(24, 26, 29),
		Mac: application.MacWindow{
			// ScreenSaver 是最高的 NSWindow 层级（1000），高于 PopUpMenu（101）、
			// Status（25，Dock 和菜单栏所在层级）及其它所有层级。这保证了
			// 遮罩始终覆盖 Dock 和菜单栏。
			WindowLevel:        application.MacWindowLevelScreenSaver,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces | application.MacWindowCollectionBehaviorStationary,
			TitleBar:            application.MacTitleBar{Hide: true, AppearsTransparent: true, FullSizeContent: true},
			DisableShadow:       true,
		},
		CloseButtonState:      application.ButtonHidden,
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
	}
}

// noticeOptions 构建休息前提醒窗口的配置项。
// 窗口透明，使用 macOS 原生半透明效果，放置在主显示器顶部居中。
func noticeOptions(a *application.App) application.WebviewWindowOptions {
	const (
		noticeWidth  = 380
		noticeHeight = 108
	)
	x, y := 0, 80
	if primary := a.Screen.GetPrimary(); primary != nil {
		x = primary.WorkArea.X + (primary.WorkArea.Width-noticeWidth)/2
		y = primary.WorkArea.Y + 80
	}
	return application.WebviewWindowOptions{
		Name:            "pm-notice",
		Title:           "",
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		Hidden:          true,
		URL:             "/#notice",
		Width:           noticeWidth,
		Height:          noticeHeight,
		X:               x,
		Y:               y,
		InitialPosition: application.WindowXY,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour:  application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop:    application.MacBackdropTranslucent,
			WindowLevel: application.MacWindowLevelFloating,
			TitleBar:    application.MacTitleBar{Hide: true, AppearsTransparent: true, FullSizeContent: true},
		},
		CloseButtonState:      application.ButtonHidden,
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
	}
}
