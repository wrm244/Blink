// 运行平台检测与快捷键修饰键命名。
//
// 用途：前端只在“展示/录制”层面需要区分平台——macOS 用 Cmd/Option，
// Windows/Linux 用 Ctrl/Alt（Ctrl 是 Wails 的 CmdOrCtrl 在 Windows 上的
// 实际解析，也是全局快捷键的通用主修饰键；Alt 对应 OptionOrAlt）。
//
// 平台来源：优先读 Wails 注入的 window._wails.environment.OS（与后端 GOOS 一致），
// 回退到 navigator.userAgent，避免任何初始化时序问题。

function detectOS(): 'windows' | 'darwin' | 'linux' | 'unknown' {
  const w = window as unknown as { _wails?: { environment?: { OS?: string } } }
  const os = w._wails?.environment?.OS
  if (os) return os as 'windows' | 'darwin' | 'linux'
  const ua = navigator.userAgent
  if (/Windows/i.test(ua)) return 'windows'
  if (/Mac/i.test(ua)) return 'darwin'
  if (/Linux/i.test(ua)) return 'linux'
  return 'unknown'
}

export const platformOS = detectOS()
export const isWindows = platformOS === 'windows'
export const isMac = platformOS === 'darwin'

// 各平台下 Wails 加速键的修饰键名称与排序（决定录制与展示顺序）。
// macOS：Cmd/Option（与 Wails accelerator 的 macOS 分支一致）。
// Windows/Linux：Ctrl/Alt/Shift。
// 注意：Windows 的 Win 键在 Wails 中需写成 "super"（"win" 不是合法 token），
// 且很多 Win+ 组合被系统占用，故录制时把 metaKey 归并到 Ctrl（见 modifierTokensFrom）。
export const MOD_NAMES: readonly string[] = isMac
  ? ['Cmd', 'Ctrl', 'Option', 'Shift']
  : ['Ctrl', 'Alt', 'Shift']

// 把键盘事件的修饰键映射为本平台 Wails 名称。
export function modifierTokensFrom(e: KeyboardEvent): string[] {
  const mods: string[] = []
  if (e.ctrlKey) mods.push('Ctrl')
  if (e.altKey) mods.push(isMac ? 'Option' : 'Alt')
  if (e.shiftKey) mods.push('Shift')
  // metaKey：macOS 上是 Cmd；Windows/Linux 上是 Win 键，归并到 Ctrl。
  if (e.metaKey) mods.push(isMac ? 'Cmd' : 'Ctrl')
  return mods
}
