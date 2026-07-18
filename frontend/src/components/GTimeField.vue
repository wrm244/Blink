<script setup lang="ts">
import { Minus, Plus } from 'lucide-vue-next';
import { computed } from 'vue';

const props = withDefaults(defineProps<{
  label: string
  desc?: string
  modelValue: number
  min: number
  max: number
  step: number
  unit?: string
  presets?: number[]
  chipLabel?: (v: number) => string
  displayValue?: (v: number) => string
}>(), {
  unit: '',
  presets: () => [],
  chipLabel: (v: number) => String(v),
  displayValue: (v: number) => String(v),
})

const emits = defineEmits<{ 'update:modelValue': [v: number] }>()

function set(v: number) {
  const clamped = Math.max(props.min, Math.min(props.max, v))
  emits('update:modelValue', clamped)
}
function dec() { set(props.modelValue - props.step) }
function inc() { set(props.modelValue + props.step) }

const atMin = computed(() => props.modelValue <= props.min)
const atMax = computed(() => props.modelValue >= props.max)
const display = computed(() => props.displayValue(props.modelValue))
</script>

<template>
  <div class="tf">
    <div class="tf__top">
      <div class="tf__title">
        <div class="tf__label">{{ label }}</div>
        <div v-if="desc" class="tf__desc">{{ desc }}</div>
      </div>
      <div class="tf__display">
        <button class="tf__step" :disabled="atMin" aria-label="−" @click="dec">
          <Minus class="size-3.5" />
        </button>
        <div class="tf__value">
          <span class="tf__num tabular-nums">{{ display }}</span>
          <span v-if="unit" class="tf__unit">{{ unit }}</span>
        </div>
        <button class="tf__step" :disabled="atMax" aria-label="+" @click="inc">
          <Plus class="size-3.5" />
        </button>
      </div>
    </div>
    <div v-if="presets.length" class="tf__chips">
      <button
        v-for="p in presets"
        :key="p"
        class="tf__chip"
        :class="{ 'tf__chip--on': modelValue === p }"
        @click="set(p)"
      >{{ chipLabel(p) }}</button>
    </div>
  </div>
</template>

<style scoped>
.tf {
  padding: 14px 16px;
  border-radius: 12px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 0.15s, background 0.15s;
}
.tf:hover { border-color: var(--accent-soft); }
.tf__top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}
.tf__title { min-width: 0; flex: 1; }
.tf__label { font-size: 13.5px; font-weight: 600; color: var(--text); }
.tf__desc { font-size: 11.5px; color: var(--text-faint); margin-top: 2px; line-height: 1.4; }
.tf__display {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.tf__step {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border-radius: 7px;
  border: 1px solid var(--glass-border);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  font-family: inherit;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}
.tf__step:hover:not(:disabled) {
  background: var(--accent-soft);
  color: var(--accent);
  border-color: var(--accent);
}
.tf__step:disabled { opacity: 0.3; cursor: not-allowed; }
.tf__value {
  display: flex;
  align-items: baseline;
  gap: 3px;
  min-width: 72px;
  justify-content: center;
}
.tf__num { font-size: 20px; font-weight: 300; color: var(--text); line-height: 1; white-space: nowrap; }
.tf__unit { font-size: 11px; color: var(--text-faint); }
.tf__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.tf__chip {
  padding: 4px 10px;
  font-size: 11.5px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--glass-border);
  border-radius: 9999px;
  cursor: pointer;
  transition: all 0.15s;
}
.tf__chip:hover {
  color: var(--text);
  border-color: var(--accent);
}
.tf__chip--on {
  color: var(--on-accent);
  background: var(--accent);
  border-color: transparent;
}
</style>
