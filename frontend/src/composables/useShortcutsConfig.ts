// 快捷键配置与冲突检测的组合式函数。

import { computed } from 'vue'
import type { Reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { Clock, Play, Settings as Settings2 } from '@lucide/vue'
import type { Settings } from '@bindings/blink/internal/config/models'
import { isMac } from '@/lib/platform'

export function useShortcutsConfig(s: Reactive<Settings>, touch: () => void) {
  const { t } = useI18n()

  // 主修饰键：macOS 用 Cmd，Windows/Linux 用 Ctrl。占位符随之切换。
  const mod = isMac ? 'Cmd' : 'Ctrl'

  // 快捷键行定义
  const shortcutRows = computed(() => [
    { key: 'shortcutStartBreak' as const, label: t('shortcuts.startBreak'), icon: Clock, value: s.shortcutStartBreak, placeholder: `${mod}+Shift+B` },
    { key: 'shortcutSkipBreak' as const, label: t('shortcuts.skipBreak'), icon: Play, value: s.shortcutSkipBreak, placeholder: `${mod}+Shift+S` },
    { key: 'shortcutPostponeBreak' as const, label: t('shortcuts.postponeBreak'), icon: Clock, value: s.shortcutPostponeBreak, placeholder: `${mod}+Shift+P` },
    { key: 'shortcutPreferences' as const, label: t('shortcuts.preferences'), icon: Settings2, value: s.shortcutPreferences, placeholder: `${mod}+Shift+,` },
  ])

  /**
   * 检测指定快捷键是否与其它行重复，返回冲突行的标签。
   * 用于在 UI 中提示重复绑定而非静默覆盖。
   */
  function conflictFor(key: keyof Settings): string | null {
    const acc = (s as any)[key] as string | undefined
    if (!acc) return null
    for (const row of shortcutRows.value) {
      if (row.key === key) continue
      if (row.value === acc) return row.label
    }
    return null
  }

  return { shortcutRows, conflictFor }
}
