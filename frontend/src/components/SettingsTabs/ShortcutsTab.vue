<script setup lang="ts">
// ShortcutsTab - 快捷键标签页：分组列表风格（与时间页/选项页/关于页同构）。
// 单面板 + 组标题 + 行分隔线；操作提示作为面板脚注（分隔线之下）。
import GlassPanel from '@/components/GlassPanel.vue'
import KeyRecorder from '@/components/KeyRecorder.vue'
import { useI18n } from 'vue-i18n'
import type { Settings } from '@bindings/blink/internal/config/models'

const { t } = useI18n()

defineProps<{
  shortcutRows: Array<{ key: keyof Settings; label: string; icon: any; value: string; placeholder: string }>
  conflictFor: (key: keyof Settings) => string | null
}>()

const emit = defineEmits<{
  'set-shortcut': [key: keyof Settings, value: string]
}>()
</script>

<template>
  <section v-show="true" class="panel-stack">
    <GlassPanel class="grp">
      <div class="grp__title">{{ t('shortcuts.title') }}</div>
      <div v-for="row in shortcutRows" :key="row.key" class="row">
        <div class="row__text">
          <div class="row__label">{{ row.label }}</div>
          <div v-if="conflictFor(row.key)" class="row__conflict">{{ t('shortcuts.conflict', { action: conflictFor(row.key) }) }}</div>
        </div>
        <KeyRecorder
          :model-value="row.value"
          :placeholder="row.placeholder"
          :prompt="t('shortcuts.prompt')"
          :clear-title="t('shortcuts.clear')"
          :conflict="!!conflictFor(row.key)"
          @update:model-value="(v: string) => emit('set-shortcut', row.key, v)"
        />
      </div>
      <div class="grp__foot">{{ t('shortcuts.hint') }}</div>
    </GlassPanel>
  </section>
</template>

<style scoped>
.panel-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: pm-fade-up 0.4s ease both;
}
/* 分组面板与标题：与其他标签页完全一致 */
.grp { padding: 16px 0 4px; }
.grp__title {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
  padding: 0 18px 12px;
}
.row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 11px 18px;
}
.row + .row { border-top: 1px solid var(--glass-border); }
.row__text { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.row__label { font-size: 13.5px; font-weight: 600; color: var(--text); }
.row__conflict { font-size: 11.5px; color: var(--phase-prebreak); }
/* 脚注提示：分隔线下的弱化说明 */
.grp__foot {
  font-size: 11.5px;
  color: var(--text-faint);
  padding: 9px 18px 12px;
  border-top: 1px solid var(--glass-border);
}
</style>
