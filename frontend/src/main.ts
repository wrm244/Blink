// 前端入口：创建 Vue 应用并挂载。
// 在挂载前应用持久化的语言和主题，避免设置加载完成前出现语言/配色闪烁。

import { createApp } from "vue"
import App from "./App.vue"
import { i18n, setLocale } from "./i18n"
import { initTheme } from "./theme"
import "./styles/index.css"

// 在挂载前应用持久化的语言 + 主题，避免设置加载时的错误语言/配色闪烁
const savedLocale = localStorage.getItem("pm:locale")
if (savedLocale === "zh-CN" || savedLocale === "en") setLocale(savedLocale)
initTheme()

createApp(App).use(i18n).mount("#app")
