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
	"os"
	"strconv"
	"sync"
	"time"
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
	mu    sync.Mutex
	data  map[string]DayStats // key = "2006-01-02"（本地日期）
	path  string
	dirty bool
	stopCh chan struct{}
}

// New 创建一个 Store。path 为空时退化为纯内存模式（不落盘），
// 便于测试。Load 会在 path 存在时读取已有数据。
// 创建后用 Start() 启动后台落盘 goroutine，用 Close() 停止并强制落盘。
func New(path string) *Store {
	s := &Store{data: make(map[string]DayStats), path: path, stopCh: make(chan struct{})}
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
	go s.flushLoop()
}

// Close 停止后台 goroutine 并强制落盘。应在前端关闭前调用。
func (s *Store) Close() {
	select {
	case <-s.stopCh:
		// 已关闭
	default:
		close(s.stopCh)
	}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = json.Unmarshal(data, &s.data)
}

// flush 将内存数据序列化并写入磁盘。仅当有未落盘的修改（dirty）时执行。
// 内部自取锁，可从任意 goroutine 调用。
func (s *Store) flush() {
	if s.path == "" {
		return
	}
	s.mu.Lock()
	if !s.dirty {
		s.mu.Unlock()
		return
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	s.dirty = false
	s.mu.Unlock()
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0o644)
}

// AddFocus 向指定日期累加专注秒数。日期取自 t 的本地日期。
// 只标记 dirty，由后台 flushLoop 定时落盘。
func (s *Store) AddFocus(t time.Time, sec int) {
	if sec <= 0 {
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
