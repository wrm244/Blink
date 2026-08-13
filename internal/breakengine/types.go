package breakengine

// Phase 是引擎当前所处的高层状态。它以字符串形式发送给前端，
// 这样 UI 无需解析数字就能切换视图。
type Phase string

const (
	// PhaseFocusing：工作周期正在倒计时，等待下一次休息。
	PhaseFocusing Phase = "focusing"
	// PhasePreBreak：休息开始前的提醒窗口。
	PhasePreBreak Phase = "prebreak"
	// PhaseShortBreak：常规短休息遮罩正在显示。
	PhaseShortBreak Phase = "shortbreak"
	// PhaseLongBreak：偶尔的较长休息遮罩正在显示。
	PhaseLongBreak Phase = "longbreak"
	// PhaseIdle：用户不活动时间超过空闲阈值；专注计时器被保持，
	// 活动恢复后重置。
	//
	// 注意：没有 PhasePaused 常量。Pause() 通过设置实时布尔值 e.paused
	// 来冻结倒计时，不改变阶段，所以用户暂停时的阶段值就是调用 Pause
	// 时的阶段（通常是 PhaseFocusing）。前端读取 State 中发出的 `paused`
	// 布尔值来检测暂停，而不是阶段字符串。
	PhaseIdle Phase = "idle"
)

// IsBreak 报告该阶段是否是正在显示的休息遮罩（短休息或长休息）。
func (p Phase) IsBreak() bool {
	return p == PhaseShortBreak || p == PhaseLongBreak
}

// State 是每次 tick 和阶段变更时发送给前端的状态快照。
// 时长以整秒表示，方便 UI 格式化。
type State struct {
	Phase           Phase `json:"phase"`
	RemainingSec    int   `json:"remainingSec"`
	TotalSec        int   `json:"totalSec"`
	BreaksUntilLong int   `json:"breaksUntilLong"`
	ShortBreakCount int   `json:"shortBreakCount"`
	Idle            bool  `json:"idle"`
	Paused          bool  `json:"paused"`
	// AutoPaused 为 true 时表示暂停由会议/媒体检测自动触发，
	// 条件消失后引擎会自动恢复；用户手动暂停时它为 false。
	AutoPaused bool `json:"autoPaused"`
	// Meeting 报告当前是否检测到会议/通话（麦克风被使用）。
	Meeting bool `json:"meeting"`
	// MediaPlaying 报告当前是否检测到媒体播放（音频输出活跃）。
	MediaPlaying bool `json:"mediaPlaying"`
}
