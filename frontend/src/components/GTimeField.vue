<script setup lang="ts">
// GTimeField - 时间设置行：标题描述在左，右侧是可输入的步进器。
// 数值点击后直接编辑（Enter/失焦确认，Escape 取消），
// +/- 按钮支持长按连发；显示统一为单一格式化文本
// （displayValue 输出如 "20 分" / "5 分"），避免数字与单位样式割裂。
import { Minus, Plus } from '@lucide/vue'
import { computed, onBeforeUnmount, ref } from 'vue'

const props = withDefaults(defineProps<{
  label: string
  desc?: string
  modelValue: number
  min: number
  max: number
  step: number
  unit?: string
  presets?: number[]
  // 预设 chip 的短标签（如 "20s" / "5m"）
  chipLabel?: (v: number) => string
  // 非编辑态的完整显示串（如 "20 分"）
  displayValue?: (v: number) => string
}>(), {
  unit: '',
  presets: () => [],
  chipLabel: (v: number) => String(v),
  displayValue: (v: number) => String(v),
})

const emits = defineEmits<{ 'update:modelValue': [v: number] }>()

/** 设置值（限制在 min/max 范围内） */
function set(v: number) {
  emits('update:modelValue', Math.max(props.min, Math.min(props.max, v)))
}

// ---- 编辑态 ----
const inputEl = ref<HTMLInputElement | null>(null)
const editing = ref(false)
const raw = ref('')
let cancelEdit = false

const shown = computed(() =>
  editing.value ? raw.value : props.displayValue(props.modelValue),
)

function onFocus() {
  editing.value = true
  raw.value = String(props.modelValue)
}
function onRaw(v: string) {
  raw.value = v.replace(/[^\d]/g, '').slice(0, 4)
}
function onBlur() {
  if (cancelEdit) {
    cancelEdit = false
    editing.value = false
    return
  }
  editing.value = false
  const n = parseInt(raw.value, 10)
  if (!Number.isNaN(n)) set(n)
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') inputEl.value?.blur()
  else if (e.key === 'Escape') { cancelEdit = true; inputEl.value?.blur() }
}

// ---- 步进按钮：点按一步，长按连发 ----
let pressTimer: number | undefined
let pressInterval: number | undefined

function bump(d: number) {
  editing.value = false
  set(props.modelValue + d)
}
function startPress(d: number) {
  bump(d)
  pressTimer = window.setTimeout(() => {
    pressInterval = window.setInterval(() => bump(d), 70)
  }, 350)
}
function stopPress() {
  clearTimeout(pressTimer)
  clearInterval(pressInterval)
}
onBeforeUnmount(stopPress)
</script>

<template>
  <div class="gtf">
    <div class="gtf__text">
      <div class="gtf__label">{{ label }}</div>
      <div v-if="desc" class="gtf__desc">{{ desc }}</div>
    </div>

    <div class="gtf__side">
      <div class="gtf__stepper">
        <button
          class="gtf__btn"
          :disabled="modelValue <= min"
          aria-label="−"
          @pointerdown.prevent="startPress(-step)"
          @pointerup="stopPress"
          @pointerleave="stopPress"
          @pointercancel="stopPress"
        >
          <Minus class="size-3" />
        </button>
        <input
          ref="inputEl"
          class="gtf__input tabular-nums"
          :value="shown"
          inputmode="numeric"
          spellcheck="false"
          @focus="onFocus"
          @blur="onBlur"
          @input="onRaw(($event.target as HTMLInputElement).value)"
          @keydown="onKeydown"
        >
        <button
          class="gtf__btn"
          :disabled="modelValue >= max"
          aria-label="+"
          @pointerdown.prevent="startPress(step)"
          @pointerup="stopPress"
          @pointerleave="stopPress"
          @pointercancel="stopPress"
        >
          <Plus class="size-3" />
        </button>
      </div>

      <div v-if="presets.length" class="gtf__chips">
        <button
          v-for="p in presets"
          :key="p"
          class="gtf__chip"
          :class="{ 'gtf__chip--on': modelValue === p }"
          @click="set(p)"
        >{{ chipLabel(p) }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.gtf {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 12px 18px;
}
.gtf__text { min-width: 0; padding-top: 2px; }
.gtf__label { font-size: 13.5px; font-weight: 600; color: var(--text); }
.gtf__desc { font-size: 11.5px; color: var(--text-faint); margin-top: 2px; line-height: 1.45; }

.gtf__side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex-shrink: 0;
}

/* 步进器：胶囊容器，内部 - 输入 + */
.gtf__stepper {
  display: flex;
  align-items: center;
  padding: 2px;
  border-radius: 9px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  transition: border-color 0.15s, background 0.15s;
}
.gtf__stepper:focus-within { border-color: var(--accent); background: var(--accent-soft); }

.gtf__btn {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border-radius: 7px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.gtf__btn:hover:not(:disabled) { background: var(--accent-soft); color: var(--accent); }
.gtf__btn:active:not(:disabled) { transform: scale(0.92); }
.gtf__btn:disabled { opacity: 0.3; cursor: not-allowed; }

/* 数值：编辑/非编辑统一为同一文本样式，消除数字与单位的大小差异 */
.gtf__input {
  width: 72px;
  padding: 4px 2px;
  text-align: center;
  font-family: inherit;
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  background: transparent;
  border: none;
  border-radius: 6px;
  outline: none;
  cursor: text;
  transition: background 0.15s;
}
.gtf__input:focus { background: var(--accent-soft); }

/* 预设 chip：小尺寸快捷值，恒定单行 */
.gtf__chips {
  display: flex;
  flex-wrap: nowrap;
  gap: 4px;
}
.gtf__chip {
  padding: 3px 8px;
  font-size: 11px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--glass-border);
  border-radius: 9999px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
}
.gtf__chip:hover {
  color: var(--text);
  border-color: var(--accent);
}
.gtf__chip--on {
  color: var(--on-accent);
  background: var(--accent);
  border-color: transparent;
}
</style>
