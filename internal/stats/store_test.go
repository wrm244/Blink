package stats

import (
	"path/filepath"
	"testing"
	"time"
)

// TestAddFocusAccumulatesSameDay 验证同一日期的多次专注累加。
func TestAddFocusAccumulatesSameDay(t *testing.T) {
	s := New("") // 纯内存模式
	t0 := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	s.AddFocus(t0, 1200)
	s.AddFocus(t0.Add(30*time.Minute), 1500)

	got := s.GetDay(2026, time.July, 15)
	if got.FocusSec != 2700 {
		t.Errorf("FocusSec=%d，应为 2700", got.FocusSec)
	}
	if got.BreakSec != 0 {
		t.Errorf("BreakSec=%d，应为 0", got.BreakSec)
	}
}

// TestAddBreakCountsSeparately 验证短/长休息分别计数。
func TestAddBreakCountsSeparately(t *testing.T) {
	s := New("")
	t0 := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	s.AddBreak(t0, 20, false)
	s.AddBreak(t0.Add(time.Hour), 20, false)
	s.AddBreak(t0.Add(2*time.Hour), 300, true)

	got := s.GetDay(2026, time.July, 15)
	if got.ShortBreaks != 2 {
		t.Errorf("ShortBreaks=%d，应为 2", got.ShortBreaks)
	}
	if got.LongBreaks != 1 {
		t.Errorf("LongBreaks=%d，应为 1", got.LongBreaks)
	}
	if got.BreakSec != 340 {
		t.Errorf("BreakSec=%d，应为 340", got.BreakSec)
	}
}

// TestGetMonthOnlyReturnsRequestedMonth 验证 GetMonth 只返回指定月份的数据。
func TestGetMonthOnlyReturnsRequestedMonth(t *testing.T) {
	s := New("")
	jul15 := time.Date(2026, time.July, 15, 10, 0, 0, 0, time.Local)
	aug20 := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.Local)
	s.AddFocus(jul15, 600)
	s.AddFocus(aug20, 900)

	jul := s.GetMonth(2026, time.July)
	if len(jul) != 1 {
		t.Fatalf("7 月返回 %d 天，应为 1 天", len(jul))
	}
	if d, ok := jul[15]; !ok || d.FocusSec != 600 {
		t.Errorf("7/15 数据错误：%v", d)
	}

	aug := s.GetMonth(2026, time.August)
	if len(aug) != 1 {
		t.Fatalf("8 月返回 %d 天，应为 1 天", len(aug))
	}
	if d, ok := aug[20]; !ok || d.FocusSec != 900 {
		t.Errorf("8/20 数据错误：%v", d)
	}
}

// TestGetDayNoRecordReturnsZero 验证无记录日期返回零值。
func TestGetDayNoRecordReturnsZero(t *testing.T) {
	s := New("")
	got := s.GetDay(2026, time.January, 1)
	if got != (DayStats{}) {
		t.Errorf("无记录日期返回非零值：%v", got)
	}
}

// TestPersistRoundTrip 验证落盘与重载的数据一致性。
func TestPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stats.json")
	s1 := New(path)
	t0 := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	s1.AddFocus(t0, 1200)
	s1.AddBreak(t0, 300, true)
	// 攒批模式下需显式 flush 才落盘。
	s1.flush()

	// 重新加载同一个文件
	s2 := New(path)
	got := s2.GetDay(2026, time.July, 15)
	if got.FocusSec != 1200 {
		t.Errorf("重载后 FocusSec=%d，应为 1200", got.FocusSec)
	}
	if got.LongBreaks != 1 {
		t.Errorf("重载后 LongBreaks=%d，应为 1", got.LongBreaks)
	}
}
