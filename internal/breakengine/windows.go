package breakengine

import (
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/platform"
)

// winWants 是状态机对"此刻屏幕上应该有什么窗口"的完整期望，
// 而不是一条增量命令。
//
// 状态机从不直接操作 AppKit：窗口的 Show/Hide/Close 和 NewWithOptions
// 内部都会调度到主线程，因此在持有引擎互斥锁时执行这些操作可能导致
// 与主线程上的绑定调用死锁。通过把所有窗口操作集中到 windowLoop 这一个
// goroutine，我们确保互斥锁不会因等待主线程而阻塞。
//
// 用"期望状态"而非"命令序列"来表达，是为了让通道满时丢弃变成无害操作：
// 早期实现把命令当队列，通道满就丢——丢一条 show 会在下一次阶段切换时
// 自愈，但丢一条 hide 会让全屏遮罩永久挡在用户屏幕上（遮罩一旦关闭不了
// 就只能强杀进程）。期望状态天然幂等：后来的期望完全覆盖先前的，所以
// 只保留最新的一条就够，而最新的一条永远不会被丢弃（见 publishWants）。
type winWants struct {
	overlays bool // 全屏休息遮罩（每块屏幕一个窗口）
	notice   bool // 休息前提醒卡片
}

// publishWants 把当前期望状态投递给窗口 goroutine，永不阻塞。
// 在引擎互斥锁下调用（e.wants 由互斥锁保护，窗口 goroutine 从不读它）。
//
// 通道容量为 1：先丢掉尚未被消费的旧期望，再放入最新的一条。两段都是
// 非阻塞的，因此即使 windowLoop 已退出（Stop 之后）也不会卡住状态机。
func (e *Engine) publishWants() {
	if e.winCh == nil {
		return
	}
	select {
	case <-e.winCh: // 覆盖未被消费的旧期望
	default:
	}
	select {
	case e.winCh <- e.wants:
	default: // 理论上不可达：刚腾出空位
	}
}

func (e *Engine) showOverlays() { e.wants.overlays = true; e.publishWants() }
func (e *Engine) hideOverlays() { e.wants.overlays = false; e.publishWants() }
func (e *Engine) showNotice()   { e.wants.notice = true; e.publishWants() }
func (e *Engine) hideNotice()   { e.wants.notice = false; e.publishWants() }

// windowState 是 windowLoop 独占的窗口句柄集合。
//
// 它必须是每代循环各自持有的一份局部状态，而不是 Engine 的字段：
// Stop() 之后旧循环还要留下来应用最后一次期望（撤掉窗口，见 windowLoop），
// 而 Start() 可能已经启动了新一代循环，两代会在一段时间内并存。
// 若句柄挂在 Engine 上，两代就会同时读写同一组字段——这是数据竞争。
// 各自持有一份后，"windowLoop 独占窗口句柄"的不变式在跨代时依然成立，
// 因此这些字段不需要任何锁。
//
// app 也随代际传入而不是读 e.app：Start 在新一代里写 e.app，而旧一代可能
// 还在收尾，读字段就是竞态。窗口句柄与它所属的 app 必须是同一代的。
type windowState struct {
	app      *application.App
	overlays []application.Window
	notice   application.Window
	// wants 是最近一次应用的期望，仅用于显示器变化的定期复核。
	wants winWants
}

// displayRecheckInterval 是休息期间复核显示器集合的间隔。
//
// 只在遮罩已经挂在屏幕上时才有工作：遮罩是按"当时"的屏幕集合创建的，
// 而长休息有 5 分钟，中途插拔外接屏完全可能。若不复核，新接入的屏幕上
// 没有遮罩，那块屏的内容对用户仍然可见——遮罩形同虚设。复核本身只是
// 一次 getAll + 窗口名集合比对，不调任何窗口 API，可以便宜地跑在 2s 级。
const displayRecheckInterval = 2 * time.Second

// windowLoop 拥有遮罩和通知窗口的控制权。它是唯一读写 ws.overlays
// 和 ws.notice 的 goroutine，因此这些字段不需要互斥锁。
//
// stopCh / winCh 由 Start 传入并只对应本代循环：本循环必须只消费自己
// 这一代的期望。若读的是 e.winCh 字段，Stop→Start 换代后本循环会开始
// 窃取新一代引擎的期望，新一代自己的 windowLoop 反而收不到——表现为遮罩
// 该显示时不显示、该关闭时关不掉。
func (e *Engine) windowLoop(stopCh chan struct{}, winCh chan winWants, app *application.App) {
	ws := windowState{app: app}
	displayCheck := time.NewTicker(displayRecheckInterval)
	defer displayCheck.Stop()
	for {
		select {
		case <-stopCh:
			// 退出前应用"停止后的期望"（Stop 已把两者置 false 并投递），
			// 否则上面的 select 可能先选中 stopCh，导致遮罩/通知窗口
			// 残留在屏幕上。只应用最后一次期望即可：它就是当前真相，
			// 且不会像逐条排队那样被丢弃。
			if w, ok := latestWants(winCh); ok {
				e.reconcileWindows(&ws, w)
			}
			return
		case w := <-winCh:
			e.reconcileWindows(&ws, w)
		case <-displayCheck.C:
			// 休息进行中：显示器集合变了就重建遮罩（仅重建，不重抢焦点）。
			if ws.wants.overlays && !e.overlaysMatch(&ws, ws.app.Screen.GetAll()) {
				e.showOverlayWindows(&ws)
			}
		}
	}
}

// latestWants 取走通道中最后一条尚未消费的期望。
func latestWants(winCh chan winWants) (winWants, bool) {
	var (
		latest winWants
		ok     bool
	)
	for {
		select {
		case w := <-winCh:
			latest, ok = w, true
		default:
			return latest, ok
		}
	}
}

// reconcileWindows 把窗口状态调整到期望状态。
//
// 不去做"期望未变则跳过"的短路：期望是完整快照，重复应用只是幂等的
// Show/Close（窗口已在显示时 Show 不做事），而一旦短路就丢掉了
// ensureOverlays 对显示器集合的复核——休息中途接入外接屏时，新屏上
// 会没有遮罩（窗口循环的定期复核只覆盖了"遮罩已在显示"的那一半）。
func (e *Engine) reconcileWindows(ws *windowState, w winWants) {
	ws.wants = w

	if w.overlays {
		e.showOverlayWindows(ws)
	} else {
		// 销毁遮罩窗口而非隐藏：隐藏会保持 WKWebView 存活，
		// 其 WebKit 渲染进程仍驻留内存。下次休息时在 ensureOverlays
		// 中重建的开销很小。
		e.closeOverlays(ws)
	}

	if w.notice {
		e.ensureNotice(ws)
		if ws.notice != nil {
			ws.notice.Show()
		}
	} else if ws.notice != nil {
		ws.notice.Close()
		ws.notice = nil
	}
}

// showOverlayWindows 创建（或复用）并显示所有显示器上的遮罩窗口。
func (e *Engine) showOverlayWindows(ws *windowState) {
	e.ensureOverlays(ws)
	// 先激活应用，再显示/聚焦窗口。Blink 是 Accessory（菜单栏）应用，
	// 遮罩弹出时焦点仍在此前的前台应用上。macOS 只对活跃应用的窗口
	// 投递 hover/点击事件，且 makeKeyWindow 在应用未激活时无效——
	// 表现为第一次点击只"激活窗口"被吃掉，跳过按钮要点第二下。
	// Activate 内部是 dispatch_async 到主队列，与后续 Show/Focus
	// 的 InvokeSync 按 FIFO 顺序执行，因此激活必然先生效。
	platform.Activate()
	for _, w := range ws.overlays {
		w.Show()
		w.Focus()
	}
}

// overlayName 根据屏幕 ID 生成遮罩窗口名称。
func overlayName(screenID string) string { return "pm-overlay-" + screenID }

// overlaysMatch 报告缓存的遮罩窗口是否仍然对应当前连接的显示器
// （数量和屏幕 ID 都相同）。
func (e *Engine) overlaysMatch(ws *windowState, screens []*application.Screen) bool {
	if len(ws.overlays) != len(screens) {
		return false
	}
	have := make(map[string]bool, len(ws.overlays))
	for _, w := range ws.overlays {
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
func (e *Engine) closeOverlays(ws *windowState) {
	for _, w := range ws.overlays {
		w.Close()
	}
	ws.overlays = nil
}

// ensureOverlays 为每个连接的显示器创建一个全屏半透明窗口
// （多显示器设置下每个屏幕都会有遮罩），在显示器集合未变化时
// 复用缓存的窗口。
func (e *Engine) ensureOverlays(ws *windowState) {
	a := ws.app
	if a == nil {
		return
	}
	screens := a.Screen.GetAll()
	if e.overlaysMatch(ws, screens) {
		return
	}
	e.closeOverlays(ws)
	for _, sc := range screens {
		screen := sc
		ws.overlays = append(ws.overlays, a.Window.NewWithOptions(overlayOptions(screen)))
	}
}

// ensureNotice 懒创建唯一的休息前提醒窗口，放置在主显示器顶部居中。
func (e *Engine) ensureNotice(ws *windowState) {
	if ws.notice != nil {
		return
	}
	a := ws.app
	if a == nil {
		return
	}
	ws.notice = a.Window.NewWithOptions(noticeOptions(a))
}
