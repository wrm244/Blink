<script setup lang="ts">
// OptionsTab - 选项标签页：长休息、音效、自动开始开关、关闭窗口行为。
import GlassPanel from '@/components/GlassPanel.vue'
import GToggle from '@/components/GToggle.vue'
import GSegmented from '@/components/GSegmented.vue'
import { Bell, Play, PanelLeftClose } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import type { Settings } from '@bindings/blink/internal/config/models'

const { t } = useI18n()

const props = defineProps<{ s: Settings }>()
const emit = defineEmits<{
  'update': [key: keyof Settings, value: boolean | string]
}>()

// 关闭窗口行为选项（仅 Windows 生效；macOS 关窗天然回到菜单栏，不显示此控件）。
const closeOptions = [
  { value: 'ask', label: t('options.closeAsk') },
  { value: 'background', label: t('options.closeBackground') },
  { value: 'quit', label: t('options.closeQuit') },
]
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
    <GlassPanel class="opt-row">
      <div class="opt-row__left">
        <span class="opt-row__label"><PanelLeftClose class="size-4" />{{ t('options.closeTitle') }}</span>
        <span class="opt-row__desc">{{ t('options.closeDesc') }}</span>
      </div>
      <GSegmented
        :options="closeOptions"
        :model-value="s.closeAction || 'ask'"
        @update:model-value="(v: string) => emit('update', 'closeAction', v)"
      />
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
