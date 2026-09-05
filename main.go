package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
	"blink/internal/config"
	"blink/internal/stats"
)

// Wails 使用 Go 的 embed 包将前端构建产物嵌入二进制文件。
//
//go:embed all:frontend/dist
var assets embed.FS

// 包级句柄，在 main 中装配，由辅助 goroutine 读取。
var (
	app    *application.App
	engine *breakengine.Engine
	statsStore *stats.Store
)

func init() {
	// 注册事件，为前端提供有类型的 JS/TS API。
	application.RegisterEvent[breakengine.State]("blink:tick")
	// blink:nav 携带一个目标 tab 名（如 "stats"），由托盘菜单发出，
	// 前端监听后切换到对应标签页。事件数据是字符串。
	application.RegisterEvent[string]("blink:nav")
}

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Printf("blink: 无法加载设置，使用默认值：%v", err)
		settings = config.Default()
	}

	// 在构建菜单前初始化托盘菜单语言。显式设置优先；空值（跟随系统）
	// 时 setMenuLanguage 内部按系统语言解析，与前端 detectSystemLocale
	// 保持一致，避免界面英文、托盘中文的分裂。
	setMenuLanguage(settings.Language)

	engine = breakengine.New(settings)
	// 注入每日统计存储（与 settings.json 同目录），引擎在阶段切换时记录。
	// 攒批落盘：后台 goroutine 每 30 秒同步，应用退出时强制 flush。
	statsStore = stats.New(stats.DefaultPath())
	statsStore.Start()
	engine.SetStatsStore(statsStore)

	app = application.New(application.Options{
		Name:        "Blink",
		Description: "Smart break reminders to ease eye strain, modelled on the 20-20-20 rule.",
		Services: []application.Service{
			application.NewService(NewBreakService(engine)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		// SingleInstance：用户再次双击二进制时不再启动新进程，而是通知
		// 已在运行的实例把设置窗口拉到前台。回调在独立 goroutine 上触发，
		// showPreferences 对 goroutine 调用者是安全的（已有窗口走 Show/Focus）。
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.blink.app",
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				showPreferences()
			},
		},
		Mac: application.MacOptions{
			// Accessory：驻留菜单栏，无 Dock 图标和应用菜单。
			ActivationPolicy: application.ActivationPolicyAccessory,
			// 窗口关闭后继续运行--托盘即应用。
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Windows: application.WindowsOptions{
			// 与 macOS 的 ApplicationShouldTerminateAfterLastWindowClosed:false 对应。
			// 不设置的话，关闭设置窗口（唯一的常驻窗口）会直接结束进程，
			// 托盘图标随之消失，计时器也停了——常驻后台的前提就没了。
			DisableQuitOnLastWindowClosed: true,
		},
	})

	buildTray()
	registerAll(settings)

	// 保持菜单栏状态和菜单标签与引擎同步。
	// 必须在 goroutine 上跑：initTrayStatus 末尾要更新托盘标签，
	// 那会 InvokeSync 回主线程。
	go initTrayStatus()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
	// 正常退出：强制落盘未写出的统计数据。
	statsStore.Close()
}
