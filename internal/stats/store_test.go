package stats

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"blink/internal/atomicfile"
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

// TestLoadNullKeepsStoreWritable 覆盖合法但危险的 JSON null：旧实现会把
// 内部 map 解成 nil，之后第一次 AddFocus/AddBreak 就会 panic。
func TestLoadNullKeepsStoreWritable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	if err := os.WriteFile(path, []byte("null"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(path)
	t0 := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)
	s.AddFocus(t0, 60)
	s.flush()

	reloaded := New(path)
	if got := reloaded.GetDay(2026, time.July, 15).FocusSec; got != 60 {
		t.Fatalf("null 统计文件恢复后写入失败：FocusSec=%d，应为 60", got)
	}
}

// TestConcurrentCloseIsSafe 固定 Store 的生命周期契约：Close 可能被并发
// 调用，不能因重复关闭 stopCh 而 panic。
func TestConcurrentCloseIsSafe(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "stats.json"))
	s.Start()

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s.Close()
		}()
	}
	close(start)
	wg.Wait()
}

// TestFlushDoesNotBlockWrites 固定 flush 的锁粒度契约：落盘（序列化 + 磁盘
// 写入 + fsync）不得发生在数据锁内。
//
// 缺陷：AddFocus/AddBreak 由引擎在阶段切换时调用，早期实现里 flush 在 s.mu
// 内写盘，于是磁盘变慢（首次写入、iCloud/Time Machine 同步目录、APFS 快照）
// 时记录调用会被 fsync 阻塞。既然 flush 的并发安全只要求「不丢新数据」，用
// writeMu 串行化写盘 + 锁内只取快照即可，无需让读路径陪着等磁盘。
func TestFlushDoesNotBlockWrites(t *testing.T) {
	s := New(filepath.Join(t.TempDir(), "stats.json"))

	writeStarted := make(chan struct{})
	releaseWrite := make(chan struct{})
	var once sync.Once
	s.writeFile = func(path string, data []byte, perm os.FileMode) error {
		once.Do(func() {
			close(writeStarted)
			<-releaseWrite
		})
		return atomicfile.WriteFile(path, data, perm)
	}

	s.AddFocus(time.Now(), 60)
	flushed := make(chan struct{})
	go func() {
		s.flush()
		close(flushed)
	}()

	<-writeStarted // 落盘进行中

	addDone := make(chan struct{})
	go func() {
		s.AddFocus(time.Now(), 30)
		close(addDone)
	}()
	select {
	case <-addDone:
	case <-time.After(time.Second):
		close(releaseWrite)
		t.Fatal("写盘期间 AddFocus 被阻塞：磁盘 IO 仍在数据锁内")
	}

	close(releaseWrite)
	<-flushed

	// 写盘期间的新数据必须仍然会在下一次 flush 落盘，不被旧快照丢掉。
	s.flush()
	if got := s.GetDay(time.Now().Year(), time.Now().Month(), time.Now().Day()).FocusSec; got != 90 {
		t.Errorf("FocusSec=%d，应为 90", got)
	}
}
