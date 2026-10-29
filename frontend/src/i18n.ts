// vue-i18n 配置：注册中英文语言包，提供语言切换与系统语言检测。

import { createI18n } from "vue-i18n"
import zhCN from "./locales/zh-CN"
import en from "./locales/en"

export type Locale = "zh-CN" | "en"

// detectSystemLocale 从浏览器/OS 偏好中选择一个受支持的语言，
// 默认为简体中文。
function detectSystemLocale(): Locale {
  const langs =
    (navigator?.languages as string[] | undefined) ??
    (navigator?.language ? [navigator.language] : [])
  for (const l of langs) {
    const low = l.toLowerCase()
    if (low.startsWith("zh")) return "zh-CN"
    if (low.startsWith("en")) return "en"
  }
  return "zh-CN"
}

export const i18n = createI18n({
  legacy: false,
  locale: detectSystemLocale(),
  fallbackLocale: "en",
  messages: {
    "zh-CN": zhCN,
    en,
  },
})

// applyLocalePreference 应用后端保存的语言偏好。空值/未知值表示跟随系统，
// 同时清除本地缓存，避免用户“放弃更改”后 localStorage 仍保留临时语言。
export function applyLocalePreference(preference: string) {
  const explicit = preference === "zh-CN" || preference === "en"
  const locale: Locale = explicit ? preference : detectSystemLocale()
  i18n.global.locale.value = locale
  if (explicit) localStorage.setItem("pm:locale", locale)
  else localStorage.removeItem("pm:locale")
  document.documentElement.lang = locale === "zh-CN" ? "zh-Hans" : "en"
}

// setLocale 切换全局语言并持久化到 localStorage。
export function setLocale(locale: Locale) {
  applyLocalePreference(locale)
}
