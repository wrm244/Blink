package config

import "testing"

// TestWithDefaultsPreservesClearedShortcuts 防止旧配置兼容逻辑吞掉用户的
// 显式选择：快捷键清空表示禁用该操作，而不是“字段缺失，应恢复默认值”。
func TestWithDefaultsPreservesClearedShortcuts(t *testing.T) {
	s := Default()
	s.ShortcutStartBreak = ""
	s.ShortcutSkipBreak = ""
	s.ShortcutPostponeBreak = ""
	s.ShortcutPreferences = ""

	got := s.withDefaults()

	if got.ShortcutStartBreak != "" || got.ShortcutSkipBreak != "" ||
		got.ShortcutPostponeBreak != "" || got.ShortcutPreferences != "" {
		t.Fatalf("withDefaults 恢复了已清空的快捷键：%+v", got)
	}
}

// TestDailyGoalZeroSurvivesDefaults 固定新字段的兼容契约：0 表示“不设每日
// 目标”，是合法值而非缺失值，不能被 withDefaults 换成默认值——否则用户
// 关掉目标后一重启就会复活。同时夹取荒谬的越界值。
func TestDailyGoalZeroSurvivesDefaults(t *testing.T) {
	s := Default()
	s.DailyGoalMin = 0
	if got := s.withDefaults().DailyGoalMin; got != 0 {
		t.Errorf("withDefaults 把关闭状态的每日目标改成了 %d", got)
	}

	s.DailyGoalMin = -30
	if got := s.withDefaults().DailyGoalMin; got != 0 {
		t.Errorf("负数每日目标应归零，得到 %d", got)
	}

	s.DailyGoalMin = maxDailyGoalMin + 1
	if got := s.withDefaults().DailyGoalMin; got != maxDailyGoalMin {
		t.Errorf("越界每日目标应夹取到 %d，得到 %d", maxDailyGoalMin, got)
	}

	s.DailyGoalMin = 120
	if got := s.withDefaults().DailyGoalMin; got != 120 {
		t.Errorf("合法每日目标被改写为 %d", got)
	}
}
