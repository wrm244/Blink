import { createApp } from "vue"
import App from "./App.vue"
import { i18n, setLocale } from "./i18n"
import { initTheme } from "./theme"
import "./styles/index.css"

// Apply persisted locale + theme before mount to avoid a flash of the wrong
// language or colour scheme while the settings load.
const savedLocale = localStorage.getItem("pm:locale")
if (savedLocale === "zh-CN" || savedLocale === "en") setLocale(savedLocale)
initTheme()

createApp(App).use(i18n).mount("#app")
