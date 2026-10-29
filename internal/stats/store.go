// Package stats 记录并持久化 Blink 每日的专注与休息时长统计。
//
// 引擎在阶段切换时调用 RecordFocus / RecordBreak 累加当天的秒数，
// 前端通过 Service 查询某月或某日的统计数据，渲染日历热力图与详细列表。
//
// 数据按本地日期（YYYY-MM-DD）聚合，持久化为 JSON：
// ~/Library/Application Support/Blink/stats.json
package stats

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"blink/internal/atomicfile"
)

// DayStats 是单日的统计聚合。
type DayStats struct {
	// FocusSec 是当日累计专注秒数。
	FocusSec int `json:"focusSec"`
	// BreakSec 是当日累计休息秒数（短休息 + 长休息）。
	BreakSec int `json:"breakSec"`
	// ShortBreaks 是当日完成的短休息次数。
	ShortBreaks int `json:"shortBreaks"`
	// LongBreaks 是当日完成的长休息次数。
	LongBreaks int `json:"longBreaks"`
}

// Store 是统计数据的内存缓存与持久化管理者，并发安全。
//
// 它被引擎持有：引擎在阶段切换时调用 Add* 方法累加数据，
// 前端通过 Service 的查询方法读取。写入攒批落盘：Add* 只标记
// dirty，由后台 flushLoop 每 30 秒同步一次到磁盘，并在 Close 时
// 强制落盘，避免每次阶段切换都同步写磁盘。
type Store struct {
	mu     sync.Mutex
	data   map[string]DayStats // key = "2006-01-02"（本地日期）
	path   string
	dirty  bool
	stopCh chan struct{}

	// lifecycleMu 串行化 Start/Close。二者可能被不同 goroutine 调用，
	// 仅靠 stopCh 的 select 检查存在并发关闭同一 channel 的竞态。
	lifecycleMu sync.Mutex
	started     bool
	closed      bool

	// writeMu 串行化落盘，使"取快照"与"写文件"可以分开：mu 只保护内存
	// 数据，磁盘 IO 在锁外进行。与 mu 的获取顺序固定为 writeMu → mu，
	// 不会死锁。
	writeMu sync.Mutex

	// writeFile 是落盘实现，默认 atomicfile.WriteFile。只在 New 里赋值、
	// 启动前由测试注入，因此无需加锁保护。
	writeFile func(path string, data []byte, perm os.FileMode) error
}

// New 创建一个 Store。path 为空时退化为纯内存模式（不落盘），
// 便于测试。Load 会在 path 存在时读取已有数据。
// 创建后用 Start() 启动后台落盘 goroutine，用 Close() 停止并强制落盘。
func New(path string) *Store {
	s := &Store{
		data:      make(map[string]DayStats),
		path:      path,
		stopCh:    make(chan struct{}),
		writeFile: atomicfile.WriteFile,
	}
	if path != "" {
		s.load()
	}
	return s
}

// Start 启动后台定时落盘 goroutine。幂等，重复调用安全。
// path 为空（纯内存模式）时是空操作。
func (s *Store) Start() {
	if s.path == "" {
		return
	}
	s.lifecycleMu.Lock()
	if s.started || s.closed {
		s.lifecycleMu.Unlock()
		return
	}
	s.started = true
	s.lifecycleMu.Unlock()
	go s.flushLoop()
}

// Close 停止后台 goroutine 并强制落盘。应在前端关闭前调用。
func (s *Store) Close() {
	s.lifecycleMu.Lock()
	if !s.closed {
		s.closed = true
		close(s.stopCh)
	}
	s.lifecycleMu.Unlock()
	s.flush()
}

const flushInterval = 30 * time.Second

// flushLoop 定时将脏数据落盘。
func (s *Store) flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.flush()
		}
	}
}

// load 从磁盘读取统计文件。文件不存在或损坏时静默忽略，
// 不阻断启动--统计是附加功能，不能让损坏的旧文件拖垮整个应用。
func (s *Store) load() {
	if s.path == "" {
		return
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	// 先解码到临时 map，再一次性替换内存状态。这样坏 JSON 不会留下
	// 半解析的数据，且合法的 JSON null 不会把 s.data 置为 nil 后在
	// AddFocus/AddBreak 写入时 panic。
	var loaded map[string]DayStats
	if err := json.Unmarshal(data, &loaded); err != nil || loaded == nil {
		return
	}
	s.mu.Lock()
	s.data = loaded
	s.mu.Unlock()
}

// flush 将内存数据序列化并写入磁盘。仅当有未落盘的修改（dirty）时执行。
// 内部自取锁，可从任意 goroutine 调用。
//
// 锁的粒度是这里的关键：
//   - writeMu 保证同一时刻只有一个 flush 在写盘。没有它，定时 flush 与
//     退出时 Close 的 flush 相遇时，后完成的写盘会用旧快照覆盖新数据，
//     而 dirty 已被清除，新数据从此不再落盘——重启后丢失统计。
//   - mu 只用来"取快照 + 清 dirty"，不做序列化和磁盘 IO。AddFocus/AddBreak
//     由引擎在阶段切换时调用，一旦与这里的磁盘 IO 抢同一把锁，引擎的 tick
//     就会被 fsync 阻塞；信号量竞争激烈时这正是"每次休息后托盘卡一下"的成因。
//
// 写盘失败时把 dirty 置回，让下一轮重试，而不是静默丢掉这批数据。
func (s *Store) flush() {
	if s.path == "" {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	snapshot, ok := s.takeSnapshot()
	if !ok {
		return
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		s.markDirty()
		return
	}
	// 原子写入：写入中途崩溃不会留下截断损坏的统计文件
	//（那会丢掉全部历史记录，load 只能静默忽略坏文件）。
	if err := s.writeFile(s.path, data, 0o644); err != nil {
		// 回置 dirty 让下一轮 flush 重试（数据仍在内存里，不必惊动用户），
		// 但要留一条日志：连续失败通常意味着磁盘满或权限问题，
		// 静默重试会让"统计一直是空的"变得无从排查。
		s.markDirty()
		log.Printf("blink/stats: 统计落盘失败，将在下一轮重试: %v", err)
	}
}

// takeSnapshot 在锁内复制一份数据副本并清除 dirty，供锁外写盘使用。
// 返回 false 表示没有待落盘的修改。
func (s *Store) takeSnapshot() (map[string]DayStats, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return nil, false
	}
	snapshot := make(map[string]DayStats, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	s.dirty = false
	return snapshot, true
}

// markDirty 在写盘失败后重新标记，保证这批数据会在下一轮 flush 重试。
func (s *Store) markDirty() {
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()
}

// AddFocus 向指定日期累加专注秒数。日期取自 t 的本地日期。
// 只标记 dirty，由后台 flushLoop 定时落盘。
// closedNow 报告 Store 是否已经 Close。取生命周期锁而不是数据锁：
// 与 Start/Close 用同一把锁串行化，避免"以为还开着、其实已 flush 完"。
func (s *Store) closedNow() bool {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	return s.closed
}

func (s *Store) AddFocus(t time.Time, sec int) {
	if sec <= 0 {
		return
	}
	// 关闭之后（退出流程中在途的引擎 tick）再写入没有意义：flushLoop 已退出，
	// 没人会把这份数据落盘。直接丢弃并留痕，好过静默留下一个永远不会保存的
	// dirty 标记。
	if s.closedNow() {
		log.Printf("blink/stats: Store 已关闭，丢弃 %d 秒记录", sec)
		return
	}
	key := t.Format(dateKeyLayout)
	s.mu.Lock()
	d := s.data[key]
	d.FocusSec += sec
	s.data[key] = d
	s.dirty = true
	s.mu.Unlock()
}

// AddBreak 向指定日期累加休息秒数，并根据 isLong 递增对应休息次数。
// 只标记 dirty，由后台 flushLoop 定时落盘。
func (s *Store) AddBreak(t time.Time, sec int, isLong bool) {
	if sec <= 0 {
		return
	}
	// 关闭之后（退出流程中在途的引擎 tick）再写入没有意义：flushLoop 已退出，
	// 没人会把这份数据落盘。直接丢弃并留痕，好过静默留下一个永远不会保存的
	// dirty 标记。
	if s.closedNow() {
		log.Printf("blink/stats: Store 已关闭，丢弃 %d 秒记录", sec)
		return
	}
	key := t.Format(dateKeyLayout)
	s.mu.Lock()
	d := s.data[key]
	d.BreakSec += sec
	if isLong {
		d.LongBreaks++
	} else {
		d.ShortBreaks++
	}
	s.data[key] = d
	s.dirty = true
	s.mu.Unlock()
}

// GetMonth 返回指定年份月份每天的统计数据。
// 返回的 map key 为该月的日期号（1-31），值是对应日的统计；
// 没有记录的日期不会出现在 map 中（调用方据此渲染空白格）。
//
// 优化：key 格式固定为 "2006-01-02"，前 7 字符是 "年-月"，
// 直接做前缀匹配后取最后 2 字符解析日期号，避免对每条记录
// 做 time.ParseInLocation（该函数涉及完整时区解析，开销较高）。
func (s *Store) GetMonth(year int, month time.Month) map[int]DayStats {
	out := make(map[int]DayStats)
	prefix := time.Date(year, month, 1, 0, 0, 0, 0, time.Local).Format("2006-01")
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.data {
		if len(k) == 10 && k[:7] == prefix {
			// k[8:10] 是日期号（"01".."31"），直接解析。
			day, err := strconv.Atoi(k[8:10])
			if err == nil && day >= 1 && day <= 31 {
				out[day] = v
			}
		}
	}
	return out
}

// GetDay 返回指定日期的统计数据。无记录时返回零值。
func (s *Store) GetDay(year int, month time.Month, day int) DayStats {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	key := t.Format(dateKeyLayout)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[key]
}

// All 返回全部统计数据的副本（主要供测试使用）。
func (s *Store) All() map[string]DayStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]DayStats, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

// dateKeyLayout 是存储键的日期格式（本地日期，无时区）。
const dateKeyLayout = "2006-01-02"
