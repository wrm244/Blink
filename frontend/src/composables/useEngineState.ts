// 引擎状态与控制逻辑的组合式函数。
// 封装与 Go 后端的交互：获取状态/设置、监听 tick 事件、
// 以及暂停/恢复/跳过/重置等用户操作。

import { Events } from '@wailsio/runtime'
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import { BreakService } from '@bindings/blink'
import type { State } from '@bindings/blink/internal/breakengine/models'
import type { Settings } from '@bindings/blink/internal/config/models'

export function useEngineState() {
  const s = reactive({} as Settings)
  const state = ref<State>({} as State)
  const dirty = ref(false)
  const saved = ref(false)
  // ready: 初始数据（设置 + 状态）加载完毕后置 true
  const ready = ref(false)
  let off: (() => void) | undefined

  // 上次保存的设置快照，用于 Discard 恢复。
  let savedSnapshot: Settings | null = null

  onMounted(async () => {
    Object.assign(s, await BreakService.GetSettings())
    savedSnapshot = { ...s }
    state.value = await BreakService.GetState()
    off = Events.On('blink:tick', (ev: { data: State }) => { state.value = ev.data })
    ready.value = true
  })
  onUnmounted(() => off?.())

  /** 标记为有未保存的更改 */
  function touch() { dirty.value = true; saved.value = false }

  /** 保存设置到后端 */
  async function save() {
    await BreakService.SaveSettings({ ...s })
    savedSnapshot = { ...s }
    dirty.value = false
    saved.value = true
  }

  /** 放弃更改，恢复到上次保存的快照 */
  function discard() {
    if (savedSnapshot) {
      Object.assign(s, savedSnapshot)
    }
    dirty.value = false
    saved.value = false
  }

  /** 完成引导流程 */
  async function completeOnboarding() {
    s.onboarded = true
    await BreakService.SaveSettings({ ...s })
    savedSnapshot = { ...s }
    await BreakService.CompleteOnboarding()
    dirty.value = false
  }

  return {
    s, state, dirty, saved, ready,
    savedSnapshot,
    touch, save, discard, completeOnboarding,
  }
}
