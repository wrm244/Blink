<script setup lang="ts">
// 休息前提醒 - 主显示器顶部滑入的提示卡片（窗口透明，见 window_options.go noticeOptions）
// 窗口四周透出桌面，卡片自身铺不透明主题底色（--notice-surface）保证可读。
// 视觉上完全对齐设置页 hero 卡片的三段结构（top → progress → bar）：
//   hero__top    图标 + 标题 | 大号细体时钟      →  top    图标芯片 + 标题 | 倒计时
//   hero__progress 通栏进度条（phase 色填充）    →  progress 同款 3px 进度条
//   hero__bar    状态点文案 | GButton 操作       →  bar    脉冲点 + 副标题 | 推迟按钮
// 样式值（字号/字重/颜色/圆角）直接取自 hero，两个窗口读起来是同一家族。
//
// 倒计时由本地按秒计时器驱动（对齐到秒边界），后端 tick 仅用于容差校正，
// 保证稳定地每秒递减，而不是跟随后端 emit 的抖动偶发跳 2 秒。
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { Eye } from '@lucide/vue'
import { BreakService } from '@bindings/blink'
import type { State } from '@bindings/blink/internal/breakengine/models'
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
let paused = false
let secTimer: number | undefined
let off: (() => void) | undefined

// 用后端值重新锚定本地倒计时。首次（seeded=false）无条件对齐；
// 之后仅在漂移 >= 2 秒时重新对齐，避免后端偶发跳秒（10→8）覆盖掉
// 我们平滑的每秒递减。本地已归零后不再被后端拉回，防止末尾 0→1→0 闪烁。
//
// 暂停时（isPaused=true）直接跟随后端冻结值：后端暂停后不再推进倒计时，
// 本地时钟若继续递减会走到 0 卡住（休息却迟迟不来），Resume 后才跳回。
function resync(sec: number, tot: number, isPaused?: boolean) {
  const wasPaused = paused
  if (typeof isPaused === 'boolean') paused = isPaused
  if (tot > 0) total.value = tot
  // 暂停：后端已冻结，本地同步显示冻结值，不递减。
  if (paused) {
    remaining.value = sec
    return
  }
  if (sec <= 0) { remaining.value = 0; return }
  // 暂停→恢复的瞬间必须无条件重锚：endAt 还停留在暂停前的时间戳，
  // 若不重锚，本地时钟会立刻按陈旧的 endAt 算出 0，界面闪 0 约 1 秒，
  // 直到下一个后端 tick 因漂移 >= 2 秒才拉回真实值。
  if (wasPaused || !seeded || Math.abs(sec - remaining.value) >= 2) {
    remaining.value = sec
    endAt = Date.now() + sec * 1000
    seeded = true
  }
}

// 对齐到下一秒边界的本地计时器：每次重新调度而非 setInterval，
// 避免回调延迟累积漂移（与休息遮罩墙钟同思路）。
// 暂停期间保持显示冻结值，不推进倒计时。
function startLocalClock() {
  const tick = () => {
    if (!paused) {
      const left = Math.max(0, Math.ceil((endAt - Date.now()) / 1000))
      remaining.value = left
    }
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
  resync(st.remainingSec, st.totalSec, st.paused)
  off = Events.On('blink:tick', (ev: { data: State }) => {
    resync(ev.data.remainingSec, ev.data.totalSec, ev.data.paused)
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
      <!-- 顶部行：图标芯片 + 标题（左），倒计时时钟（右）——与 hero__top 同构 -->
      <div class="top">
        <div class="brand">
          <div class="icon" aria-hidden="true">
            <Eye class="size-4" />
          </div>
          <div class="title">{{ t('notice.title') }}</div>
        </div>
        <div class="clock">
          <span class="num tabular-nums">{{ remaining }}</span>
          <span class="unit">{{ t('notice.secondsShort') }}</span>
        </div>
      </div>

      <!-- 进度条：与 hero__progress 同构，phase-prebreak 填充，1s linear 平滑推进 -->
      <div class="progress" aria-hidden="true">
        <div class="progress__fill" :style="{ transform: `scaleX(${progress})` }" />
      </div>

      <!-- 底部行：脉冲状态点 + 副标题（左），推迟按钮（右）——与 hero__bar 同构 -->
      <div class="bar">
        <span class="dot" aria-hidden="true" />
        <span class="sub">{{ t('notice.sub') }}</span>
        <GButton variant="glass" size="sm" class="postpone" @click="postpone">
          {{ t('notice.postpone') }}
        </GButton>
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

/* 卡片布局与圆角（16px = rounded-2xl，与设置页面板一致）。
   背景自绘不透明主题底色（--notice-surface）：窗口是全透明的，
   单靠玻璃填充在深色主题下只有 7% 白，在 Windows（无原生毛玻璃）
   上等于没有背景。边框/顶部高光/阴影仍由 .glass-strong 提供。 */
.notice {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  border-radius: 16px;
  padding: 12px 16px 10px;
  box-sizing: border-box;
  gap: 9px;
  color: var(--text);
  user-select: none;
  animation: pm-fade-up 0.35s ease both;
  background: var(--notice-surface);
}

/* ---- 顶部行：与 hero__top 同构 ---- */
.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
/* 图标芯片：28px 圆角方块（mini 版 hero logo 位），phase-prebreak 着色 */
.icon {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 9px;
  background: var(--accent-soft);
  border: 1px solid var(--glass-border);
  color: var(--phase-prebreak);
}
.title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
  letter-spacing: -0.005em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 倒计时时钟：与 hero__time 同款排版（细字重 + tabular-nums），颜色用中性
   --text（phase 色只出现在图标/进度条/状态点上，与 hero 的用法一致）。 */
.clock {
  display: flex;
  align-items: baseline;
  gap: 3px;
  flex-shrink: 0;
}
.num {
  font-size: 26px;
  font-weight: 250;
  line-height: 1;
  color: var(--text);
  letter-spacing: -0.02em;
}
.unit {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-faint);
}

/* ---- 进度条：与 hero__progress 同构（3px 细线 + scaleX 推进） ---- */
.progress {
  height: 3px;
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

/* ---- 底部行：与 hero__bar 同构 ---- */
.bar {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}
/* 脉冲状态点：与保存栏 dirty 圆点同款 pm-pulse 动画，传递"正在进行"的紧迫感 */
.dot {
  width: 7px;
  height: 7px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--phase-prebreak);
  animation: pm-pulse 2.4s ease-in-out infinite;
}
.sub {
  font-size: 11.5px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.postpone {
  margin-left: auto;
  flex-shrink: 0;
}
</style>
