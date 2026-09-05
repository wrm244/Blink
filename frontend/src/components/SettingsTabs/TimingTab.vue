<script setup lang="ts">
// TimingTab - 计时设置标签页：预设方案 + 计时字段网格。
import GlassPanel from '@/components/GlassPanel.vue'
import GTimeField from '@/components/GTimeField.vue'
import { Check } from '@lucide/vue'
import type { Settings } from '@bindings/blink/internal/config/models'
import type { TimingField, PresetProfile } from '@/composables/useTimingConfig'

defineProps<{
  presetProfiles: PresetProfile[]
  activePreset: PresetProfile['id']
  timingFields: TimingField[]
  s: Settings
}>()

const emit = defineEmits<{
  'apply-preset': [p: PresetProfile]
  'set-num': [field: keyof Settings, v: number]
}>()
</script>

<template>
  <section v-show="true" class="panel-stack">
    <!-- 预设方案 -->
    <GlassPanel class="presets">
      <div class="presets__head">
        <div class="presets__title">{{ $t('timing.presets') }}</div>
      </div>
      <div class="presets__grid">
        <button
          v-for="p in presetProfiles"
          :key="p.id"
          class="preset"
          :class="{ 'preset--on': activePreset === p.id, 'preset--custom': p.id === 'custom' }"
          @click="emit('apply-preset', p)"
        >
          <div class="preset__name">{{ p.name }}</div>
          <div v-if="p.desc" class="preset__desc">{{ p.desc }}</div>
          <Check v-if="activePreset === p.id" class="preset__check size-3.5" />
        </button>
      </div>
    </GlassPanel>

    <!-- 计时参数列表（分组列表 + 行内下拉选择） -->
    <GlassPanel class="timing-list">
      <div class="timing-list__title">{{ $t('timing.parameters') }}</div>
      <GTimeField
        v-for="f in timingFields"
        :key="f.key"
        class="timing-list__row"
        :label="f.label"
        :desc="f.desc"
        :unit="f.unit"
        :min="f.min"
        :max="f.max"
        :step="f.step"
        :presets="f.presets"
        :model-value="(s as any)[f.key] ?? 0"
        :chip-label="f.formatChip"
        :display-value="f.formatValue"
        @update:model-value="(v: number) => emit('set-num', f.key, v)"
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

.presets { padding: 16px 18px; }
.presets__head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.presets__title {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
}
.presets__grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}
.preset {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 12px;
  text-align: left;
  font-family: inherit;
  color: var(--text-muted);
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.18s;
  overflow: hidden;
}
.preset:hover { color: var(--text); border-color: var(--accent); }
.preset--on {
  color: var(--on-accent);
  background: var(--accent);
  border-color: transparent;
}
.preset__name { font-size: 13px; font-weight: 600; }
.preset__desc { font-size: 11px; color: var(--text-faint); line-height: 1.35; }
.preset--on .preset__desc { color: var(--on-accent); opacity: 0.85; }
.preset__check {
  position: absolute;
  top: 8px;
  right: 8px;
  color: var(--on-accent);
}

.timing-list {
  padding: 16px 0 4px;
}
/* 与 .presets 的标题对齐：面板水平 padding 18px，标题下方间距 12px */
.timing-list__title {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
  padding: 0 18px 12px;
}
/* 行间细分隔线（子组件根节点带父级 scoped 属性，可直接选中） */
.timing-list__row + .timing-list__row {
  border-top: 1px solid var(--glass-border);
}
@media (max-width: 720px) {
  .presets__grid { grid-template-columns: 1fr 1fr; }
}
</style>
