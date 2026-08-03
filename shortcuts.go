package main

import (
	"log"
	"sync"

	"blink/internal/config"
)

// rebindMu 串行化快捷键重新注册，防止密集的设置保存交替执行 UnregisterAll/Register。
var rebindMu sync.Mutex

// registerAll 从设置中绑定四个全局快捷键，替换之前已绑定的所有快捷键。
// 可从任何 goroutine 调用：app 运行前绑定被延迟，运行后调度到主线程
//（这就是已持有主线程的调用者--SaveSettings 绑定调用--将其分发到独立
// goroutine 的原因）。
func registerAll(s config.Settings) {
	rebindMu.Lock()
	defer rebindMu.Unlock()
	if app == nil {
		return
	}
	_ = app.GlobalShortcut.UnregisterAll()

	bind := func(acc string, fn func()) {
		if acc == "" {
			return
		}
		if err := app.GlobalShortcut.Register(acc, fn); err != nil {
			log.Printf("blink: 快捷键 %q 注册失败：%v", acc, err)
		}
	}
	bind(s.ShortcutStartBreak, func() { engine.StartBreakNow() })
	bind(s.ShortcutSkipBreak, func() { engine.SkipBreak() })
	bind(s.ShortcutPostponeBreak, func() { engine.PostponeBreak() })
	bind(s.ShortcutPreferences, func() { showPreferences() })
}
