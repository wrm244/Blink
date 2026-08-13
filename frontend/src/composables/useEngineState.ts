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
  // editVersion 每次编辑递增，用于识别保存期间是否又发生了新的编辑
  // （async save 竞态：保存请求在途时用户继续改，返回后不能把 dirty 复位）。
  let editVersion = 0

  onMounted(async () => {
    Object.assign(s, await BreakService.GetSettings())
    savedSnapshot = { ...s }
    state.value = await BreakService.GetState()
    off = Events.On('blink:tick', (ev: { data: State }) => { state.value = ev.data })
    ready.value = true
  })
  onUnmounted(() => off?.())

  /** 标记为有未保存的更改 */
  function touch() { dirty.value = true; saved.value = false; editVersion++ }

  /** 保存设置到后端。返回是否成功。 */
  async function save(): Promise<boolean> {
    const v = editVersion
    try {
      await BreakService.SaveSettings({ ...s })
    } catch (err) {
      // 保存失败：保持 dirty，让用户看到"未保存"状态后重试。
      console.error('保存设置失败', err)
      return false
    }
    // 保存期间用户又改了别的字段：不要吞掉新编辑，保持 dirty。
    if (v !== editVersion) return true
    savedSnapshot = { ...s }
    dirty.value = false
    saved.value = true
    return true
  }

  /**
   * 同步后端已经写入的设置变更（并非用户在本页面编辑的结果）。
   *
   * 用于关闭确认弹窗这类"后端直接落盘"的路径：必须连同 savedSnapshot
   * 一起更新，否则这里的 s 仍是旧值，用户之后改别的设置一保存，
   * 就会把后端刚存好的值原样覆盖回去。也正因为它不是未保存的编辑，
   * 不能置 dirty。
   */
  function applyExternal(patch: Partial<Settings>) {
    Object.assign(s, patch)
    if (savedSnapshot) savedSnapshot = { ...savedSnapshot, ...patch }
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
    try {
      await BreakService.SaveSettings({ ...s })
    } catch (err) {
      // 保存失败：保持未完成状态，用户可重试。
      console.error('保存引导设置失败', err)
      return
    }
    savedSnapshot = { ...s }
    await BreakService.CompleteOnboarding()
    dirty.value = false
  }

  return {
    s, state, dirty, saved, ready,
    savedSnapshot,
    touch, save, discard, completeOnboarding, applyExternal,
  }
}
