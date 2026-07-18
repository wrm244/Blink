// Theme controller: applies a light/dark/system colour scheme to the document
// and keeps "system" in sync with the OS appearance.

export type Theme = "system" | "light" | "dark"

let media: MediaQueryList | null = null
let systemListener: (() => void) | null = null

// resolvedTheme returns the concrete appearance ("light" | "dark") for a theme
// preference, resolving "system" against the OS preference.
function resolvedTheme(theme: Theme): "light" | "dark" {
  if (theme === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
  }
  return theme
}

// applyTheme toggles the `dark` class on <html> to switch the CSS variable set,
// and (for "system") subscribes to OS appearance changes so the app follows the
// system in real time.
export function applyTheme(theme: Theme) {
  // Normalise: empty/unknown values fall back to "system".
  const t: Theme = theme === "light" || theme === "dark" ? theme : "system"

  const resolved = resolvedTheme(t)
  document.documentElement.classList.toggle("dark", resolved === "dark")

  // Rebind the system listener only when following the system.
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

// initTheme applies the last-used theme early at startup (before the settings
// load), to avoid a flash of the wrong theme.
export function initTheme() {
  const saved = (localStorage.getItem("pm:theme") as Theme | null) ?? "system"
  applyTheme(saved)
}
