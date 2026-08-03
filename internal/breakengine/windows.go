package breakengine

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/platform"
)

// windowCmd 是从状态机 goroutine 发送给窗口控制 goroutine 的消息（非阻塞）。
// 状态机从不直接操作 AppKit：窗口的 Show/Hide/Close 和 NewWithOptions
// 内部都会调度到主线程，因此在持有引擎互斥锁时执行这些操作可能导致
// 与主线程上的绑定调用死锁。通过将所有窗口操作集中到这个单一 goroutine，
// 我们确保互斥锁不会因等待主线程而阻塞。
type windowCmd int

const (
	cmdShowOverlays windowCmd = iota
	cmdHideOverlays
	cmdShowNotice
	cmdHideNotice
)

// sendCmd 非阻塞地入队一个窗口命令。在引擎互斥锁下调用；
// 丢弃命令是无害的，因为后续命令会覆盖前面的（例如 hide 跟在 show 后面会胜出）。
func (e *Engine) sendCmd(c windowCmd) {
	select {
	case e.cmdCh <- c:
	default:
	}
}

func (e *Engine) showOverlays() { e.sendCmd(cmdShowOverlays) }
func (e *Engine) hideOverlays() { e.sendCmd(cmdHideOverlays) }
func (e *Engine) showNotice()   { e.sendCmd(cmdShowNotice) }
func (e *Engine) hideNotice()   { e.sendCmd(cmdHideNotice) }

// windowLoop 拥有遮罩和通知窗口的控制权。它是唯一读写 e.overlays
// 和 e.notice 的 goroutine，因此这些字段不需要互斥锁。
func (e *Engine) windowLoop() {
	for {
		select {
		case <-e.stopCh:
			// 退出前排空 Stop 入队的命令（如 hide overlays），
			// 否则上面的 select 可能先选中 stopCh，导致遮罩/通知窗口
			// 残留在屏幕上。
			for {
				select {
				case c := <-e.cmdCh:
					e.execWindowCmd(c)
				default:
					return
				}
			}
		case c := <-e.cmdCh:
			e.execWindowCmd(c)
		}
	}
}

// execWindowCmd 执行单个窗口命令。
func (e *Engine) execWindowCmd(c windowCmd) {
	switch c {
	case cmdShowOverlays:
		e.ensureOverlays()
		// 先激活应用，再显示/聚焦窗口。Blink 是 Accessory（菜单栏）应用，
		// 遮罩弹出时焦点仍在此前的前台应用上。macOS 只对活跃应用的窗口
		// 投递 hover/点击事件，且 makeKeyWindow 在应用未激活时无效——
		// 表现为第一次点击只"激活窗口"被吃掉，跳过按钮要点第二下。
		// Activate 内部是 dispatch_async 到主队列，与后续 Show/Focus
		// 的 InvokeSync 按 FIFO 顺序执行，因此激活必然先生效。
		platform.Activate()
		for _, w := range e.overlays {
			w.Show()
			w.Focus()
		}
	case cmdHideOverlays:
		// 销毁遮罩窗口而非隐藏：隐藏会保持 WKWebView 存活，
		// 其 WebKit 渲染进程仍驻留内存。下次休息时在 ensureOverlays
		// 中重建的开销很小。
		e.closeOverlays()
	case cmdShowNotice:
		e.ensureNotice()
		if e.notice != nil {
			e.notice.Show()
		}
	case cmdHideNotice:
		if e.notice != nil {
			e.notice.Close()
			e.notice = nil
		}
	}
}

// overlayName 根据屏幕 ID 生成遮罩窗口名称。
func overlayName(screenID string) string { return "pm-overlay-" + screenID }

// overlaysMatch 报告缓存的遮罩窗口是否仍然对应当前连接的显示器
// （数量和屏幕 ID 都相同）。
func (e *Engine) overlaysMatch(screens []*application.Screen) bool {
	if len(e.overlays) != len(screens) {
		return false
	}
	have := make(map[string]bool, len(e.overlays))
	for _, w := range e.overlays {
		have[w.Name()] = true
	}
	for _, s := range screens {
		if !have[overlayName(s.ID)] {
			return false
		}
	}
	return true
}

// closeOverlays 关闭并清空所有遮罩窗口。
func (e *Engine) closeOverlays() {
	for _, w := range e.overlays {
		w.Close()
	}
	e.overlays = nil
}

// ensureOverlays 为每个连接的显示器创建一个全屏半透明窗口
// （多显示器设置下每个屏幕都会有遮罩），在显示器集合未变化时
// 复用缓存的窗口。
func (e *Engine) ensureOverlays() {
	a := e.app_()
	if a == nil {
		return
	}
	screens := a.Screen.GetAll()
	if e.overlaysMatch(screens) {
		return
	}
	e.closeOverlays()
	for _, sc := range screens {
		screen := sc
		e.overlays = append(e.overlays, a.Window.NewWithOptions(overlayOptions(screen)))
	}
}

// ensureNotice 懒创建唯一的休息前提醒窗口，放置在主显示器顶部居中。
func (e *Engine) ensureNotice() {
	if e.notice != nil {
		return
	}
	a := e.app_()
	if a == nil {
		return
	}
	e.notice = a.Window.NewWithOptions(noticeOptions(a))
}
