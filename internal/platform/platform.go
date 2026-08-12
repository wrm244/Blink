// Package platform 暴露 break engine 需要但 Wails 未封装的少量宿主 OS 行为：
// 用户最后操作键盘/鼠标至今的时长、播放短促的系统提示音、把窗口强行提到
// 前台，以及注册/注销开机自启（登录项）。
//
// 具体实现按构建约束拆分为三套：
//   - *_darwin.go  —— CGo 调用 AppKit / ApplicationServices / CoreServices
//   - *_windows.go —— 通过 user32 / kernel32 / winmm / 注册表调用 Win32 API
//   - *_other.go   —— 其余平台的空操作桩
//
// 在没有原生 API 的平台上优雅降级：IdleSeconds 返回 0（视为"始终活跃"），
// PlaySound 与窗口相关函数为空操作。
package platform
