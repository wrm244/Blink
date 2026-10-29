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
  // loaded: 设置是否真的从后端读到过。读失败时不允许保存——
  // 那会把空对象写回后端，抹掉用户的全部配置。
  const loaded = ref(false)
  let off: (() => void) | undefined

  // 上次保存的设置快照，用于 Discard 恢复。
  let savedSnapshot: Settings | null = null
  // editVersion 每次编辑递增，用于识别保存期间是否又发生了新的编辑
  // （async save 竞态：保存请求在途时用户继续改，返回后不能把 dirty 复位）。
  let editVersion = 0

  // loadSettings 拉取后端设置并建立保存快照，失败时返回 false。
  //
  // fillMissingOnly：只填充 s 里尚未存在的字段，不覆盖已有值。
  // 保存前补读时必须用这个模式——用户可能已经在界面上改了几项，
  // 补读是为了凑齐“没改过的字段”，而不是把用户的编辑冲掉。
  async function loadSettings(fillMissingOnly = false): Promise<boolean> {
    try {
      const got = await BreakService.GetSettings()
      if (fillMissingOnly) {
        const target = s as Record<string, unknown>
        for (const [k, v] of Object.entries(got)) {
          if (!(k in target)) target[k] = v
        }
        // 快照取后端真实值（而不是合并后的 s）：Discard 的语义是“回到上次
        // 保存的状态”，用户本次的编辑应该被丢掉。
        savedSnapshot = { ...got }
      } else {
        Object.assign(s, got)
        savedSnapshot = { ...s }
      }
      loaded.value = true
      return true
    } catch (err) {
      console.error('读取设置失败', err)
      return false
    }
  }

  onMounted(async () => {
    // 关键：无论加载成功与否，最后都必须把 ready 置 true。
    // 设置窗口是 Hidden 创建、只在 Settings.vue 的 watch(ready) 里调
    // ShowWindow 显示。早期实现这里直接 await 且无兜底，任一绑定调用
    // reject（后端未就绪、IPC 超时）就让 ready 永远为 false——用户点
    // 托盘里的"设置"后窗口根本不出现，也没有任何报错提示。
    if (!(await loadSettings())) {
      // 只重试一次：多半是后端 Service 尚未绑定完成的瞬时失败。
      await new Promise((r) => setTimeout(r, 400))
      await loadSettings()
    }
    try {
      state.value = await BreakService.GetState()
    } catch (err) {
      // 状态读不到不影响设置界面：tick 事件到达后会自然填充。
      console.error('读取引擎状态失败', err)
    }
    off = Events.On('blink:tick', (ev: { data: State }) => { state.value = ev.data })
    ready.value = true
  })
  onUnmounted(() => off?.())

  /** 标记为有未保存的更改 */
  function touch() { dirty.value = true; saved.value = false; editVersion++ }

  /** 保存设置到后端。返回是否成功。 */
  async function save(): Promise<boolean> {
    // 设置从未成功读到过时拒绝保存：此时 s 只是空对象，写回后端会把
    // 用户的真实配置抹成零值。先补一次读取（只补缺失字段，保住用户
    // 已做的编辑），后续重试才有意义。
    if (!loaded.value && !(await loadSettings(true))) return false
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
    s, state, dirty, saved, ready, loaded,
    touch, save, discard, completeOnboarding, applyExternal,
  }
}
