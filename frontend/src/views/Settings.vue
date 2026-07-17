<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { BreakService } from '../../bindings/pocketmind'
import type { Settings } from '../../bindings/pocketmind/internal/config/models'
import type { State } from '../../bindings/pocketmind/internal/breakengine/models'

// `s` mirrors the backend settings; edits are local until Save is pressed.
// Settings is a generated interface, so we build a plain typed object rather
// than constructing a class.
const s = reactive({} as Settings)
const state = ref<State>({} as State)
const dirty = ref(false)
const saved = ref(false)
let off: (() => void) | undefined

onMounted(async () => {
  Object.assign(s, await BreakService.GetSettings())
  state.value = await BreakService.GetState()
  off = Events.On('pm:tick', (ev: { data: State }) => {
    state.value = ev.data
  })
})
onUnmounted(() => off?.())

function touch() {
  dirty.value = true
  saved.value = false
}

async function save() {
  // Spread to a plain object so the value matches the Settings interface
  // exactly (the reactive proxy type is not directly assignable to the param).
  await BreakService.SaveSettings({ ...s })
  dirty.value = false
  saved.value = true
}

function startBreak() { BreakService.StartBreakNow() }
function skipBreak() { BreakService.SkipBreak() }
function postponeBreak() { BreakService.PostponeBreak() }
function reset() { BreakService.Reset() }
function togglePause() {
  if (state.value.paused) BreakService.Resume()
  else BreakService.Pause()
}

const phaseLabel = computed(() => {
  switch (state.value.phase) {
    case 'focusing': return `Focusing · ${fmt(state.value.remainingSec)} to break`
    case 'prebreak': return `Break in ${state.value.remainingSec}s`
    case 'shortbreak': return `Short break · ${fmt(state.value.remainingSec)}`
    case 'longbreak': return `Long break · ${fmt(state.value.remainingSec)}`
    case 'paused': return 'Paused'
    case 'idle': return 'Paused (inactive)'
    default: return '-'
  }
})
function fmt(sec: number): string {
  const m = Math.floor(sec / 60)
  const ss = sec % 60
  return `${m}:${String(ss).padStart(2, '0')}`
}
</script>

<template>
  <div class="prefs">
    <header class="prefs__header">
      <div>
        <h1 class="prefs__title">PocketMind</h1>
        <p class="prefs__status">{{ phaseLabel }}</p>
      </div>
      <div class="prefs__actions">
        <button class="btn btn--ghost" @click="startBreak">Break now</button>
        <button class="btn btn--ghost" @click="togglePause">{{ state.paused ? 'Resume' : 'Pause' }}</button>
      </div>
    </header>

    <section class="card">
      <h2 class="card__title">Timing</h2>
      <div class="grid">
        <label class="field">
          <span>Focus duration <em>minutes</em></span>
          <input type="number" min="1" max="180" v-model.number="s.focusDurationMin" @input="touch" />
        </label>
        <label class="field">
          <span>Short break <em>seconds</em></span>
          <input type="number" min="5" max="600" v-model.number="s.shortBreakDurationSec" @input="touch" />
        </label>
        <label class="field">
          <span>Long break <em>minutes</em></span>
          <input type="number" min="1" max="60" v-model.number="s.longBreakDurationMin" @input="touch" />
        </label>
        <label class="field">
          <span>Long break every <em>breaks</em></span>
          <input type="number" min="1" max="20" v-model.number="s.longBreakInterval" @input="touch" />
        </label>
        <label class="field">
          <span>Pre-break warning <em>seconds</em></span>
          <input type="number" min="0" max="120" v-model.number="s.preBreakWarningSec" @input="touch" />
        </label>
        <label class="field">
          <span>Idle pause after <em>minutes</em></span>
          <input type="number" min="1" max="60" v-model.number="s.idleThresholdMin" @input="touch" />
        </label>
      </div>
    </section>

    <section class="card">
      <h2 class="card__title">Options</h2>
      <label class="toggle">
        <input type="checkbox" v-model="s.enableLongBreaks" @change="touch" />
        <span>Enable occasional long breaks</span>
      </label>
      <label class="toggle">
        <input type="checkbox" v-model="s.soundEnabled" @change="touch" />
        <span>Play a chime when a break ends</span>
      </label>
    </section>

    <section class="card">
      <h2 class="card__title">Shortcuts</h2>
      <div class="grid">
        <label class="field">
          <span>Start break now</span>
          <input type="text" v-model="s.shortcutStartBreak" @input="touch" placeholder="Cmd+Shift+B" />
        </label>
        <label class="field">
          <span>Skip break</span>
          <input type="text" v-model="s.shortcutSkipBreak" @input="touch" placeholder="Cmd+Shift+S" />
        </label>
        <label class="field">
          <span>Postpone break</span>
          <input type="text" v-model="s.shortcutPostponeBreak" @input="touch" placeholder="Cmd+Shift+P" />
        </label>
        <label class="field">
          <span>Show preferences</span>
          <input type="text" v-model="s.shortcutPreferences" @input="touch" placeholder="Cmd+Shift+," />
        </label>
      </div>
      <p class="card__hint">Accelerators use Wails syntax, e.g. <code>Cmd+Shift+B</code>.</p>
    </section>

    <footer class="prefs__footer">
      <span class="prefs__saved" :class="{ 'is-on': saved }">Saved</span>
      <div>
        <button class="btn btn--ghost" @click="skipBreak">Skip</button>
        <button class="btn btn--ghost" @click="postponeBreak">Postpone</button>
        <button class="btn btn--ghost" @click="reset">Reset cycle</button>
        <button class="btn btn--primary" :disabled="!dirty" @click="save">Save</button>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.prefs {
  min-height: 100vh;
  padding: 40px 36px 28px;
  box-sizing: border-box;
  color: #e9edf5;
  font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Inter', sans-serif;
}
.prefs__header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 26px;
}
.prefs__title {
  margin: 0;
  font-size: 26px;
  font-weight: 600;
  letter-spacing: -0.01em;
}
.prefs__status {
  margin: 6px 0 0;
  font-size: 13px;
  color: rgba(233, 237, 245, 0.55);
  font-variant-numeric: tabular-nums;
}
.prefs__actions {
  display: flex;
  gap: 8px;
}
.card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 14px;
  padding: 18px 18px 20px;
  margin-bottom: 16px;
}
.card__title {
  margin: 0 0 14px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: rgba(233, 237, 245, 0.5);
}
.grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px 18px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field span {
  font-size: 13px;
  color: rgba(233, 237, 245, 0.75);
}
.field em {
  font-style: normal;
  color: rgba(233, 237, 245, 0.4);
}
.field input {
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: #e9edf5;
  padding: 9px 11px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.15s ease;
}
.field input:focus {
  border-color: rgba(91, 157, 255, 0.7);
}
.toggle {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: rgba(233, 237, 245, 0.85);
  padding: 6px 0;
  cursor: pointer;
}
.toggle input {
  accent-color: #5b9dff;
}
.card__hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: rgba(233, 237, 245, 0.4);
}
.card__hint code {
  background: rgba(255, 255, 255, 0.08);
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 11px;
}
.prefs__footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}
.prefs__saved {
  font-size: 13px;
  color: #5bd98a;
  opacity: 0;
  transition: opacity 0.2s ease;
}
.prefs__saved.is-on {
  opacity: 1;
}
.btn {
  padding: 8px 16px;
  font-size: 13px;
  border-radius: 9px;
  cursor: pointer;
  border: 1px solid rgba(255, 255, 255, 0.12);
  transition: background 0.15s ease, opacity 0.15s ease;
}
.btn--ghost {
  background: rgba(255, 255, 255, 0.06);
  color: rgba(233, 237, 245, 0.85);
}
.btn--ghost:hover {
  background: rgba(255, 255, 255, 0.12);
}
.btn--primary {
  background: #5b9dff;
  color: #06101f;
  border-color: transparent;
  font-weight: 600;
  margin-left: 8px;
}
.btn--primary:hover {
  background: #74abff;
}
.btn--primary:disabled {
  opacity: 0.4;
  cursor: default;
}
</style>
