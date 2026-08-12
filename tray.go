package main

import (
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	tray                  *application.SystemTray
	statusItem            *application.MenuItem
	menuItemTakeBreak     *application.MenuItem
	menuItemSkip          *application.MenuItem
	menuItemPostpone      *application.MenuItem
	menuItemPauseResume   *application.MenuItem
	menuItemStart         *application.MenuItem
	menuItemReset         *application.MenuItem
	menuItemStats         *application.MenuItem
	menuItemPreferences   *application.MenuItem
	menuItemQuit          *application.MenuItem
	// menuLang 持有当前托盘菜单语言。它被 trayStatusLoop（以及运行在
	// 主线程的菜单点击处理器）读取，被 setMenuLanguage（SaveSettings
	// 在 goroutine 上分发）写入。使用 atomic.Value 而非裸字符串，
	// 这样竞态检测器保持安静，trayStatusLoop 中的读取永远不会看到
	// 半写值。
	menuLang atomic.Value
)

func init() {
	menuLang.Store("zh-CN")
}

// menuLangString 以普通字符串形式加载当前菜单语言。
func menuLangString() string {
	return menuLang.Load().(string)
}

// 托盘菜单字符串，按语言分组。保存在 Go 中是因为菜单在 macOS 侧原生构建，
// 而非在 webview 中。
var trayStrings = map[string]map[string]string{
	"zh-CN": {
		"takeBreak":   "立即开始休息",
		"skip":        "跳过休息",
		"postpone":    "推迟休息",
		"pauseResume": "暂停 / 继续",
		"startFocus":  "开始专注",
		"reset":       "重置周期",
		"stats":       "统计…",
		"preferences": "设置…",
		"quit":        "退出 Blink",
	},
	"en": {
		"takeBreak":   "Take a break now",
		"skip":        "Skip break",
		"postpone":    "Postpone break",
		"pauseResume": "Pause / Resume",
		"startFocus":  "Start focusing",
		"reset":       "Reset cycle",
		"stats":       "Statistics…",
		"preferences": "Preferences…",
		"quit":        "Quit Blink",
	},
}

// tr 返回指定键在当前语言下的托盘菜单文本。
func tr(key string) string {
	if m, ok := trayStrings[menuLangString()]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return trayStrings["en"][key]
}

// buildTray 创建菜单栏状态项及其下拉菜单。
func buildTray() {
	menu := application.NewMenu()

	// 禁用的首项，兼作实时状态读数。
	statusItem = menu.Add("Blink")
	statusItem.SetEnabled(false)

	menu.AddSeparator()
	menuItemTakeBreak = menu.Add(tr("takeBreak"))
	menuItemTakeBreak.OnClick(func(*application.Context) { engine.StartBreakNow() })
	menuItemSkip = menu.Add(tr("skip"))
	menuItemSkip.OnClick(func(*application.Context) { engine.SkipBreak() })
	menuItemPostpone = menu.Add(tr("postpone"))
	menuItemPostpone.OnClick(func(*application.Context) { engine.PostponeBreak() })
	menu.AddSeparator()
	menuItemPauseResume = menu.Add(tr("pauseResume"))
	menuItemPauseResume.OnClick(func(*application.Context) { togglePause() })
	menuItemStart = menu.Add(tr("startFocus"))
	menuItemStart.OnClick(func(*application.Context) { engine.Start() })
	menuItemReset = menu.Add(tr("reset"))
	menuItemReset.OnClick(func(*application.Context) { engine.Reset() })
	menu.AddSeparator()
	menuItemStats = menu.Add(tr("stats"))
	menuItemStats.OnClick(func(*application.Context) { go showStats() })
	menuItemPreferences = menu.Add(tr("preferences"))
	// 走 goroutine：菜单回调运行在主线程，而 showPreferences 在窗口已存在
	// 时走 Show/Focus（内部 InvokeSync 回主线程），直接调用会自锁。
	// 与 tray_platform_windows.go 的 OnDoubleClick 同理。
	menuItemPreferences.OnClick(func(*application.Context) { go showPreferences() })
	menu.AddSeparator()
	menuItemQuit = menu.Add(tr("quit"))
	menuItemQuit.OnClick(func(*application.Context) { app.Quit() })

	tray = app.SystemTray.New()
	tray.SetLabel("Blink")
	tray.SetTooltip("Blink")
	tray.SetMenu(menu)
	applyTrayPlatform(tray, menu)
}

// setMenuLanguage 为指定语言重建托盘菜单项标签。
func setMenuLanguage(lang string) {
	menuLang.Store(lang)
	if menuItemTakeBreak != nil {
		menuItemTakeBreak.SetLabel(tr("takeBreak"))
		menuItemSkip.SetLabel(tr("skip"))
		menuItemPostpone.SetLabel(tr("postpone"))
		menuItemPauseResume.SetLabel(tr("pauseResume"))
		menuItemStart.SetLabel(tr("startFocus"))
		menuItemReset.SetLabel(tr("reset"))
		menuItemStats.SetLabel(tr("stats"))
		menuItemPreferences.SetLabel(tr("preferences"))
		menuItemQuit.SetLabel(tr("quit"))
	}
}

// pendingNav 持有"待打开的目标 tab"，由托盘菜单设置，前端在
// mount/ready 后通过 GetPendingNav() 读取并清空。空串表示无待处理导航。
var pendingNav string

// showStats 打开设置窗口并切换到统计标签页。
//
// 复用 showPreferences 的窗口创建/显示逻辑，再设置 pendingNav 为 "stats"，
// 前端在 ready 后读取它来切换 tab。这样首次创建窗口（前端尚未 mount，
// 无法接收事件）和已存在窗口两种情况都能可靠地切到统计页。
//
// 走 goroutine：与 showPreferences 同理，菜单回调运行在主线程，
// 而窗口已存在时 showPreferences 走 Show/Focus（InvokeSync 回主线程）。
func showStats() {
	pendingNav = "stats"
	showPreferences()
}

// togglePause 根据当前暂停状态切换暂停/继续。
func togglePause() {
	if engine.GetState().Paused {
		engine.Resume()
	} else {
		engine.Pause()
	}
}
