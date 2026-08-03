package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
	"blink/internal/config"
)

// Wails 使用 Go 的 embed 包将前端构建产物嵌入二进制文件。
//
//go:embed all:frontend/dist
var assets embed.FS

// 包级句柄，在 main 中装配，由辅助 goroutine 读取。
var (
	app    *application.App
	engine *breakengine.Engine
)

func init() {
	// 注册事件，为前端提供有类型的 JS/TS API。
	application.RegisterEvent[breakengine.State]("blink:tick")
}

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Printf("blink: 无法加载设置，使用默认值：%v", err)
		settings = config.Default()
	}

	// 在构建菜单前从已保存的设置初始化托盘菜单语言。
	if settings.Language == "zh-CN" || settings.Language == "en" {
		menuLang.Store(settings.Language)
	}

	engine = breakengine.New(settings)

	app = application.New(application.Options{
		Name:        "Blink",
		Description: "Smart break reminders to ease eye strain, modelled on the 20-20-20 rule.",
		Services: []application.Service{
			application.NewService(NewBreakService(engine)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Accessory：驻留菜单栏，无 Dock 图标和应用菜单。
			ActivationPolicy: application.ActivationPolicyAccessory,
			// 窗口关闭后继续运行--托盘即应用。
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	buildTray()
	registerAll(settings)

	// 保持菜单栏状态和菜单标签与引擎同步。
	go trayStatusLoop()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
