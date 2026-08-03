<script setup lang="ts">
// ShortcutsTab - 快捷键标签页：快捷键录制与冲突检测。
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
    <GlassPanel v-for="row in shortcutRows" :key="row.key" class="shortcut-row">
      <div class="shortcut-row__left">
        <span class="shortcut-row__label"><component :is="row.icon" class="size-4" />{{ row.label }}</span>
        <span v-if="conflictFor(row.key)" class="shortcut-row__conflict">{{ t('shortcuts.conflict', { action: conflictFor(row.key) }) }}</span>
      </div>
      <KeyRecorder
        :model-value="row.value"
        :placeholder="row.placeholder"
        :prompt="t('shortcuts.prompt')"
        :clear-title="t('shortcuts.clear')"
        :conflict="!!conflictFor(row.key)"
        @update:model-value="(v: string) => emit('set-shortcut', row.key, v)"
      />
    </GlassPanel>
    <p class="hint">{{ t('shortcuts.hint') }}</p>
  </section>
</template>

<style scoped>
.panel-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: pm-fade-up 0.4s ease both;
}
.shortcut-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 13px 20px; }
.shortcut-row__left { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.shortcut-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.shortcut-row__conflict { font-size: 11.5px; color: var(--phase-prebreak); }
.hint { font-size: 12px; color: var(--text-faint); padding: 4px 6px; }
</style>
