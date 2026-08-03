// Package platform 暴露 break engine 需要但 Wails 未封装的少量宿主 OS 行为：
// 用户最后操作键盘/鼠标至今的时长，以及播放短促的系统提示音。
//
// 具体实现按构建约束拆分（见 idle_darwin.go / sound_darwin.go
// 及其 *_other.go 桩文件）。在没有原生 API 的平台上优雅降级：
// IdleSeconds 返回 0（视为"始终活跃"），PlaySound 为空操作。
package platform
