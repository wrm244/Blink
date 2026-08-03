<script setup lang="ts">
// 休息前提醒 - 主显示器顶部滑入的纤细提示（窗口透明，见 windows.go noticeOptions）。
// 视觉上复用设置页的 Slate 玻璃设计系统：glass-strong 面板 + phase-prebreak 强调色 +
// 与 hero 同源的大号细体数字 + GButton，消除与设置页的割裂感。
// 倒计时由本地按秒计时器驱动（对齐到秒边界），后端 tick 仅用于容差校正，
// 保证稳定地每秒递减，而不是跟随后端 emit 的抖动偶发跳 2 秒。
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '@bindings/blink'
import GButton from '@/components/GButton.vue'

const { t } = useI18n()
const remaining = ref(0)
const total = ref(1)

// 已过去时间占比 (0 → 1)，与设置页 hero 进度条同义，让提示条随时间平滑填充。
const progress = computed(() =>
  total.value > 0 ? Math.max(0, Math.min(1, 1 - remaining.value / total.value)) : 0,
)

// endAt：本地倒计时目标结束时间戳(ms)。由后端 remainingSec 锚定，
// 之后本地按真实秒边界递减，不再依赖后端每秒 emit 的稳定性。
let endAt = 0
let seeded = false
let secTimer: number | undefined
let off: (() => void) | undefined

// 用后端值重新锚定本地倒计时。首次（seeded=false）无条件对齐；
// 之后仅在漂移 >= 2 秒时重新对齐，避免后端偶发跳秒（10→8）覆盖掉
// 我们平滑的每秒递减。本地已归零后不再被后端拉回，防止末尾 0→1→0 闪烁。
function resync(sec: number, tot: number) {
  if (tot > 0) total.value = tot
  if (sec <= 0) { remaining.value = 0; return }
  if (!seeded || Math.abs(sec - remaining.value) >= 2) {
    remaining.value = sec
    endAt = Date.now() + sec * 1000
    seeded = true
  }
}

// 对齐到下一秒边界的本地计时器：每次重新调度而非 setInterval，
// 避免回调延迟累积漂移（与休息遮罩墙钟同思路）。
function startLocalClock() {
  const tick = () => {
    const left = Math.max(0, Math.ceil((endAt - Date.now()) / 1000))
    remaining.value = left
    secTimer = window.setTimeout(tick, 1000 - (Date.now() % 1000))
  }
  tick()
}

onMounted(async () => {
  // 本窗口是透明窗（window_options.go 的 noticeOptions 设 BackgroundTypeTransparent），
  // 但全局样式给 html/body 铺了 var(--bg-from)，会把透明窗染成暗色背景。
  // 这里把 html/body 拉回透明，只显示玻璃卡片本身，四周透出桌面。
  // 仅作用于本 WebView 进程，不影响设置页 / 休息遮罩窗口。
  document.documentElement.style.background = 'transparent'
  document.body.style.background = 'transparent'

  const st = await BreakService.GetState()
  resync(st.remainingSec, st.totalSec)
  off = Events.On('blink:tick', (ev: { data: { remainingSec: number; totalSec: number } }) => {
    resync(ev.data.remainingSec, ev.data.totalSec)
  })
  startLocalClock()
})
onUnmounted(() => {
  off?.()
  if (secTimer !== undefined) clearTimeout(secTimer)
})

function postpone() { BreakService.PostponeBreak() }
</script>

<template>
  <div class="notice-wrap">
    <div class="notice glass-strong">
      <!-- 强调条：4px 竖条，使用休息前阶段色，与设置页 cycle__dot / phaseColor 同源。 -->
      <div class="accent" aria-hidden="true" />

      <div class="content">
        <div class="row">
          <!-- 倒计时簇：大号细数字 + 安静单位，与设置页 hero 时钟同源排版。 -->
          <div class="count tabular-nums">
            <span class="num">{{ remaining }}</span>
            <span class="unit">{{ t('notice.secondsShort') }}</span>
          </div>

          <!-- 文案：标题 + 副标题，字号/颜色与设置页面板一致。 -->
          <div class="body">
            <div class="title">{{ t('notice.title') }}</div>
            <div class="sub">{{ t('notice.sub') }}</div>
          </div>

          <!-- 推迟：幽灵式按钮，复用设置页 GButton(ghost) 组件。 -->
          <GButton variant="ghost" size="sm" class="postpone" @click="postpone">
            {{ t('notice.postpone') }}
          </GButton>
        </div>

        <!-- 进度条：与设置页 hero__progress 同源，随时间平滑填充（1s linear）。 -->
        <div class="progress" aria-hidden="true">
          <div class="progress__fill" :style="{ transform: `scaleX(${progress})` }" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notice-wrap {
  width: 100%;
  height: 100%;
  padding: 6px;
  box-sizing: border-box;
}

/* 卡片本体交给 .glass-strong 提供表面（半透明填充 + 发丝边框 + 顶部高光），
   这里只负责布局与圆角，不覆盖其背景/边框。 */
.notice {
  display: flex;
  align-items: stretch;
  width: 100%;
  height: 100%;
  border-radius: 16px; /* rounded-2xl，与设置页面板一致 */
  overflow: hidden;
  color: var(--text);
  user-select: none;
  animation: pm-fade-up 0.35s ease both;
}

/* 强调条：4px 竖条紧贴左边缘，phase-prebreak 色。 */
.accent {
  width: 4px;
  flex-shrink: 0;
  background: var(--phase-prebreak);
}

.content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  padding: 10px 14px;
  box-sizing: border-box;
}

.row {
  display: flex;
  align-items: center;
  gap: 14px;
}

/* 倒计时簇：数字与设置页 hero 时钟同族（细字重 + tabular-nums），
   让两个屏幕读起来相关。 */
.count {
  display: flex;
  align-items: baseline;
  gap: 3px;
  flex-shrink: 0;
  color: var(--phase-prebreak);
}
.num {
  font-size: 34px;
  font-weight: 300;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
.unit {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-faint);
}

.body {
  flex: 1;
  min-width: 0;
}
.title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.005em;
}
.sub {
  font-size: 11px;
  color: var(--text-faint);
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.postpone {
  flex-shrink: 0;
}

/* 进度条：与设置页 hero__progress 同源，2px 细线，phase-prebreak 填充。 */
.progress {
  height: 2px;
  border-radius: 9999px;
  background: var(--track);
  overflow: hidden;
}
.progress__fill {
  height: 100%;
  width: 100%;
  border-radius: 9999px;
  background: var(--phase-prebreak);
  transform-origin: left center;
  transition: transform 1s linear;
  will-change: transform;
}
</style>
