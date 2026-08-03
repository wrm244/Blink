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

// setLocale 切换全局语言并持久化到 localStorage。
export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
  localStorage.setItem("pm:locale", locale)
  // 同步文档语言属性以支持无障碍访问
  document.documentElement.lang = locale === "zh-CN" ? "zh-Hans" : "en"
}
