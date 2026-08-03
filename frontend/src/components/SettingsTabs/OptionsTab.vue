<script setup lang="ts">
// OptionsTab - 选项标签页：长休息、音效、自动开始开关。
import GlassPanel from '@/components/GlassPanel.vue'
import GToggle from '@/components/GToggle.vue'
import { Bell, Play } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { Settings } from '@bindings/blink/internal/config/models'

const { t } = useI18n()

const props = defineProps<{ s: Settings }>()
const emit = defineEmits<{
  'update': [key: keyof Settings, value: boolean]
}>()
</script>

<template>
  <section v-show="true" class="panel-stack">
    <GlassPanel class="opt-row">
      <div class="opt-row__left">
        <span class="opt-row__label"><Play class="size-4" />{{ t('options.autoStart') }}</span>
        <span class="opt-row__desc">{{ t('options.autoStartDesc') }}</span>
      </div>
      <GToggle :model-value="s.autoStart" @update:model-value="(v: boolean) => emit('update', 'autoStart', v)" />
    </GlassPanel>
    <GlassPanel class="opt-row">
      <div class="opt-row__left">
        <span class="opt-row__label"><Bell class="size-4" />{{ t('options.sound') }}</span>
        <span class="opt-row__desc">{{ t('options.soundDesc') }}</span>
      </div>
      <GToggle :model-value="s.soundEnabled" @update:model-value="(v: boolean) => emit('update', 'soundEnabled', v)" />
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
.opt-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 15px 20px; }
.opt-row__left { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.opt-row__label { font-size: 14px; font-weight: 500; display: flex; align-items: center; gap: 8px; color: var(--text); }
.opt-row__desc { font-size: 12px; color: var(--text-faint); }
</style>
