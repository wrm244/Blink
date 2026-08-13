//go:build !darwin

package platform

// AudioActivity 在没有进程级音频检测 API 的平台上返回 false,false
// （不触发会议/媒体自动暂停）。
func AudioActivity() (meeting, media bool) { return false, false }
