import { createApp } from "vue"
import App from "./App.vue"
import { i18n, setLocale } from "./i18n"
import "./styles/index.css"

// Apply the persisted language if the backend reported one; otherwise the
// system-detected default from i18n.ts is already active.
const saved = localStorage.getItem("pm:locale")
if (saved === "zh-CN" || saved === "en") setLocale(saved)

createApp(App).use(i18n).mount("#app")
