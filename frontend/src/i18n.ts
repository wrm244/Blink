import { createI18n } from "vue-i18n"
import zhCN from "./locales/zh-CN"
import en from "./locales/en"

export type Locale = "zh-CN" | "en"

// detectSystemLocale picks a supported locale from the browser/OS preference,
// defaulting to Simplified Chinese.
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

export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
  localStorage.setItem("pm:locale", locale)
  // Keep the document language attribute in sync for accessibility.
  document.documentElement.lang = locale === "zh-CN" ? "zh-Hans" : "en"
}
