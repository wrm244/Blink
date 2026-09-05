package breakengine

import (
	"testing"
	"time"

	"blink/internal/platform"
)

// 本文件量化本次优化的实际收益，供回归对比：
//   - 两个开关都关闭时跳过 CGo 音频探测，每秒省下一次跨语言调用；
//   - 状态未变的 emit 被去重，省下一次快照序列化与事件广播。
//
// 跳过的那次调用成本由 Disabled / Enabled 两个基准的差值给出。

// BenchmarkCheckExternalSuspendDisabled 测量开关全关时的每轮开销：
// 此时 checkExternalSuspend 应在触碰音频子系统之前就返回。
func BenchmarkCheckExternalSuspendDisabled(b *testing.B) {
	e := newTestEngine()
	e.settings.PauseOnMeeting = false
	e.settings.PauseOnMedia = false
	startEngine(e)
	e.audioProbe = platform.AudioActivity

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.checkExternalSuspend()
	}
}

// BenchmarkCheckExternalSuspendEnabled 测量开关打开时的每轮开销，
// 与上一条对比即得 CGo 探测本身的成本。
func BenchmarkCheckExternalSuspendEnabled(b *testing.B) {
	e := newTestEngine()
	e.settings.PauseOnMeeting = true
	e.settings.PauseOnMedia = true
	startEngine(e)
	e.audioProbe = platform.AudioActivity

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.checkExternalSuspend()
	}
}

// BenchmarkTickSteady 测量稳态每秒 tick 的开销（专注中，不触发阶段切换）。
// 用于确认把外部检测合并进主循环后，单次唤醒的成本没有退化。
func BenchmarkTickSteady(b *testing.B) {
	e := newTestEngine()
	startEngine(e)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每轮把阶段拉回"专注中且远未到期"，避免触发阶段切换后
		// 的测量对象变成 startPreBreak / startBreak。
		e.phaseEnd = now.Add(20 * time.Minute)
		e.tick(now.Add(time.Duration(i) * 10 * time.Millisecond))
	}
}

// BenchmarkEmitUnchanged 测量冻结态（暂停）下的 emit 开销：状态未变时
// 应在比对快照后直接返回。
func BenchmarkEmitUnchanged(b *testing.B) {
	e := newTestEngine()
	startEngine(e)
	e.Pause()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.emit()
	}
}

// BenchmarkEmitChanged 测量状态每次都变时的 emit 开销，与上一条对比
// 即得去重省下的部分。注意 app 为 nil（测试环境没有 Wails 应用），
// 因此这里量的是"快照计算 + 去重判断"，不含真正投递给 webview 的成本。
func BenchmarkEmitChanged(b *testing.B) {
	e := newTestEngine()
	startEngine(e)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.phaseEnd = now.Add(time.Duration(i) * time.Second)
		e.emit()
	}
}
