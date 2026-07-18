package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
	"blink/internal/config"
)

// Wails uses Go's embed package to ship the frontend build inside the binary.
//go:embed all:frontend/dist
var assets embed.FS

// Package-level handles wired up in main and read by the helper goroutines.
var (
	app    *application.App
	engine *breakengine.Engine
)

func init() {
	// Registering the event gives the frontend a typed JS/TS API for it.
	application.RegisterEvent[breakengine.State]("blink:tick")
}

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Printf("blink: could not load settings, using defaults: %v", err)
		settings = config.Default()
	}

	// Seed the tray menu locale from saved settings before the menu is built.
	if settings.Language == "zh-CN" || settings.Language == "en" {
		menuLang = settings.Language
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
			// Accessory: live in the menu bar without a Dock icon or app menu.
			ActivationPolicy: application.ActivationPolicyAccessory,
			// Keep running when windows close - the tray is the app.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	buildTray()
	registerAll(settings)

	// Keep the menu-bar status and menu label in sync with the engine.
	go trayStatusLoop()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
