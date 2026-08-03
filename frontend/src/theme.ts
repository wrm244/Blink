// 主题控制器：为文档应用浅色/深色/系统配色方案，
// 并在"系统"模式下实时跟随 OS 外观变化。

export type Theme = "system" | "light" | "dark"

let media: MediaQueryList | null = null
let systemListener: (() => void) | null = null

// resolvedTheme 返回主题偏好对应的具体外观（"light" | "dark"），
// 将"system"解析为 OS 偏好。
function resolvedTheme(theme: Theme): "light" | "dark" {
  if (theme === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
  }
  return theme
}

// applyTheme 切换 <html> 上的 `dark` 类以切换 CSS 变量集，
// 并在"system"模式下订阅 OS 外观变化以实时跟随系统。
export function applyTheme(theme: Theme) {
  // 规范化：空/未知值回退到 "system"
  const t: Theme = theme === "light" || theme === "dark" ? theme : "system"

  const resolved = resolvedTheme(t)
  document.documentElement.classList.toggle("dark", resolved === "dark")

  // 仅在跟随系统时重新绑定监听器
  if (media && systemListener) {
    media.removeEventListener("change", systemListener)
    media = null
    systemListener = null
  }
  if (t === "system") {
    media = window.matchMedia("(prefers-color-scheme: dark)")
    systemListener = () => applyTheme("system")
    media.addEventListener("change", systemListener)
  }

  localStorage.setItem("pm:theme", t)
}

// initTheme 在启动早期（设置加载前）应用上次使用的主题，
// 避免错误主题的闪烁。
export function initTheme() {
  const saved = (localStorage.getItem("pm:theme") as Theme | null) ?? "system"
  applyTheme(saved)
}
