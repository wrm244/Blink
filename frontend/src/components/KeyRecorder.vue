<script setup lang="ts">
// KeyRecorder - 快捷键录制组件。
// 监听真实按键并发出 Wails 加速键字符串（如 "Cmd+Shift+B"）。
// macOS 修饰键命名与 Wails accelerator.String() 的平台分支一致：
// Cmd, Ctrl, Option, Shift。
import { Eraser, Keyboard } from '@lucide/vue'
import { computed, onUnmounted, ref } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  prompt?: string
  clearTitle?: string
  disabled?: boolean
  conflict?: boolean
}>(), {
  placeholder: '-',
  prompt: '…',
  clearTitle: 'Clear',
  disabled: false,
  conflict: false,
})

const emits = defineEmits<{ 'update:modelValue': [v: string] }>()

const recording = ref(false)

// 修饰键名称，按 Wails 序列化顺序排列
const MOD_NAMES = ['Cmd', 'Ctrl', 'Option', 'Shift'] as const
// 命名键集合
const NAMED_KEYS = new Set([
  'backspace', 'tab', 'return', 'enter', 'escape', 'space', 'delete',
  'home', 'end', 'page up', 'page down', 'left', 'right', 'up', 'down',
  'f1', 'f2', 'f3', 'f4', 'f5', 'f6', 'f7', 'f8', 'f9', 'f10',
  'f11', 'f12', 'f13', 'f14', 'f15', 'f16', 'f17', 'f18', 'f19', 'f20',
])

/** 将键盘事件归一化为 Wails 加速键名称，无效键返回 null */
function normalizeKey(e: KeyboardEvent): string | null {
  const raw = e.key
  // 修饰键本身不构成快捷键，等待真实按键
  if (['Meta', 'Alt', 'Shift', 'Control'].includes(raw)) return null
  const lower = raw.toLowerCase()
  // F 键（浏览器给出 "F1"、"F2"…）直接映射到 Wails 的 f1..f20
  if (/^f\d{1,2}$/.test(lower)) return lower
  if (raw === 'ArrowLeft') return 'left'
  if (raw === 'ArrowRight') return 'right'
  if (raw === 'ArrowUp') return 'up'
  if (raw === 'ArrowDown') return 'down'
  if (raw === 'PageUp') return 'page up'
  if (raw === 'PageDown') return 'page down'
  if (lower === ' ' || lower === 'spacebar') return 'space'
  if (lower === '+') return 'plus'
  if (NAMED_KEYS.has(lower)) return lower
  // 单个可打印字符（字母、数字、标点）。修饰键和控制键有多字符名称，
  // 已在上方捕获，所以这里的单字符键是真实字符键。
  if (lower.length === 1) return lower
  return null
}

/** 从键盘事件中提取修饰键列表 */
function modsFrom(e: KeyboardEvent): string[] {
  const mods: string[] = []
  if (e.metaKey) mods.push('Cmd')
  if (e.ctrlKey) mods.push('Ctrl')
  if (e.altKey) mods.push('Option')
  if (e.shiftKey) mods.push('Shift')
  return mods
}

/** 录制中的 keydown 处理器 */
function onKeydown(e: KeyboardEvent) {
  if (!recording.value) return
  e.preventDefault()
  e.stopPropagation()
  // 吞掉系统对未处理键的"按键"声（如 Delete）
  if (e.key === 'Delete') e.stopImmediatePropagation()

  if (e.key === 'Escape') {
    recording.value = false
    return
  }
  const mods = modsFrom(e)
  const key = normalizeKey(e)
  if (key === null) return
  if (mods.length === 0) return
  // 按 Wails 序列化顺序排列修饰键
  const ordered = MOD_NAMES.filter(m => mods.includes(m))
  emits('update:modelValue', [...ordered, key].join('+'))
  recording.value = false
}

/** 开始录制 */
function start() {
  if (props.disabled) return
  recording.value = true
  window.addEventListener('keydown', onKeydown, true)
}
/** 停止录制 */
function stop() {
  recording.value = false
  window.removeEventListener('keydown', onKeydown, true)
}

// 显示值：有值则显示值，否则显示占位符
const display = computed(() => (props.modelValue && props.modelValue.length ? props.modelValue : props.placeholder))

onUnmounted(() => {
  if (recording.value) stop()
})
</script>

<template>
  <div class="kr" :class="{ 'kr--recording': recording, 'kr--conflict': conflict }" @click="recording ? stop() : start()">
    <Keyboard class="kr__icon size-3.5" />
    <span v-if="recording" class="kr__prompt">{{ prompt }}</span>
    <span v-else class="kr__keys">{{ display }}</span>
    <button
      v-if="modelValue && !recording"
      class="kr__clear"
      :disabled="disabled"
      :title="clearTitle"
      @click.stop="emits('update:modelValue', '')"
    >
      <Eraser class="size-3" />
    </button>
  </div>
</template>

<style scoped>
.kr {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-width: 150px;
  padding: 8px 10px;
  font-size: 13px;
  font-family: ui-monospace, "SF Mono", Menlo, monospace;
  color: var(--text);
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  cursor: pointer;
  user-select: none;
  transition: border-color 0.15s, background 0.15s, color 0.15s;
}
.kr:hover { border-color: var(--accent); }
.kr--recording {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent);
}
.kr--conflict { border-color: var(--phase-prebreak); }
.kr--recording.kr--conflict { border-color: var(--phase-prebreak); }
.kr__icon { color: var(--text-faint); flex-shrink: 0; }
.kr--recording .kr__icon { color: var(--accent); }
.kr__keys { letter-spacing: 0.01em; white-space: nowrap; }
.kr__prompt { color: var(--accent); opacity: 0.75; font-style: italic; margin-left: 2px; }
.kr__clear {
  display: grid;
  place-items: center;
  margin-left: auto;
  padding: 3px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-faint);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.kr__clear:hover { background: var(--accent-soft); color: var(--text); }
.kr__clear:disabled { opacity: 0.3; cursor: not-allowed; }
</style>
