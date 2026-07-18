# Blink

[English](./README.md) | [简体中文](./README.zh-CN.md)

一款 macOS 菜单栏护眼提醒应用，基于 [20-20-20 规则](https://www.aoa.org/healthy-eyes/caring-for-your-eyes/protecting-your-eyes/20-20-20-rule)：每用屏 20 分钟，看 20 英尺（约 6 米）外的地方 20 秒。

使用 [Wails v3](https://v3.wails.io)（Go + Vue 3 + TypeScript）、[Tailwind CSS v4](https://tailwindcss.com) 和 [vue-i18n](https://vue-i18n.intlify.dev) 构建，支持中英双语。

## 功能

- **首次引导** — 首次启动进入设置界面，完成配置并点击"开始专注"后才开始计时。
- **菜单栏图标** 实时显示距下次休息的倒计时。
- **专注 → 休息预警 → 休息** 循环，频繁的短休息穿插偶尔的长休息。
- **全屏休息覆盖层** 覆盖所有显示器（多屏感知），显示大号倒计时；休息结束时播放提示音。覆盖层位于系统窗口最高层级，覆盖一切。
- **休息预警** 在屏幕变暗前给你时间收尾当前任务。
- **空闲感知** — 离开时专注计时器暂停并重置，只在真正工作时触发休息。支持系统睡眠后恢复。
- **全局快捷键** 开始 / 跳过 / 推迟 / 打开设置。
- **完全可配置** 通过设置界面（时长滑块、选项、快捷键、关于），持久化到 `~/Library/Application Support/Blink/settings.json`。
- **双语界面**（简体中文 / English）：默认跟随系统语言，可在设置中切换；原生托盘菜单同步本地化。

## 设计

Blink 采用**石板灰设计体系** — 单一中性强调色，无渐变，降饱和的语义色。休息覆盖层是一片沉静的深石板灰背景，倒计时数字是绝对主角。设计目标是稳重而不打扰：应用融入你的工作流，只在需要休息时发声。

## 快速开始

```bash
wails3 dev        # 热重载开发
wails3 build      # 生产构建 -> bin/blink
wails3 task package   # -> bin/blink.app（临时签名）
```

首次启动会出现设置窗口。选择语言并点击**开始专注**（也可先调整时长）。之后托盘图标显示实时倒计时。点击菜单栏图标（或按 `Cmd+Shift+,`）打开设置，按 `Cmd+Shift+B` 立即休息。

## 架构

| 包 | 职责 |
| --- | --- |
| `internal/breakengine` | 计时状态机（阶段、空闲、睡眠重置、休息窗口、启停） |
| `internal/config` | 设置结构体 + JSON 持久化（含 `onboarded`、`autoStart`、`language`） |
| `internal/platform` | CGo：macOS 系统空闲时间（`CGEventSource`）和 `NSSound` 提示音；其他平台为空实现 |
| `main.go` / `tray.go` / `shortcuts.go` / `prefs.go` / `breakservice.go` | 应用接入：托盘+菜单（i18n）、全局快捷键、设置窗口、前端调用的 Wails 服务（引导门控 + 引擎控制） |
| `frontend/src` | Vue 3 SPA — 哈希路由按窗口选择 Settings / BreakOverlay / PreBreakNotice；玻璃面板组件在 `components/`，语言包在 `locales/` |

> 注：会议/通话检测和媒体播放检测（自动暂停功能）不在当前版本范围内。

## 许可

(c) 2026, Blink
