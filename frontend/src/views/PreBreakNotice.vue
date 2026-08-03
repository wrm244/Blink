<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '@bindings/blink'

const { t } = useI18n()
const remaining = ref(0)
let off: (() => void) | undefined

onMounted(async () => {
  const st = await BreakService.GetState()
  remaining.value = st.remainingSec
  off = Events.On('blink:tick', (ev: { data: { remainingSec: number } }) => {
    remaining.value = ev.data.remainingSec
  })
})
onUnmounted(() => off?.())

function postpone() { BreakService.PostponeBreak() }
</script>

<template>
  <div class="notice-wrap">
    <div class="notice">
      <!-- 左侧强调条：macOS 通知风格的视觉锚点，
           使用休息前阶段色。替代了旧的脉冲圆点 + 阴影方块。 -->
      <div class="accent" aria-hidden="true" />

      <!-- 倒计时是核心：大号细数字（与休息遮罩排版一致），
           单位作为安静上下文紧跟旁边。 -->
      <div class="countdown tabular-nums">
        <span class="num">{{ remaining }}</span>
        <span class="unit">{{ t('notice.secondsShort') }}</span>
      </div>

      <!-- 标题 + 副标题：纯文本，无卡片背板。 -->
      <div class="body">
        <div class="title">{{ t('notice.title') }}</div>
        <div class="sub">{{ t('notice.sub') }}</div>
      </div>

      <!-- 推迟：幽灵式文字按钮。无玻璃卡片、无边框 -
           仅一个悬停时变暖的安静交互。 -->
      <button class="postpone" @click="postpone">
        {{ t('notice.postpone') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
/*
 * 休息前提醒 - 休息开始前约 10 秒在主显示器顶部滑入的纤细提示。
 * 窗口本身透明，使用 macOS 原生半透明效果（见 windows.go noticeOptions），
 * 所以 .notice 外壳仅添加淡色底 + 发丝边框，将内容置于系统毛玻璃之上。
 * 此处不用 backdrop-filter：在窗口原生模糊上叠加 CSS 模糊
 * 会让旧版看起来像贴在桌面上的厚重卡片。
 */
.notice-wrap {
  width: 100%;
  height: 100%;
  padding: 6px;
  box-sizing: border-box;
}
.notice {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  height: 100%;
  padding: 0 14px 0 0;
  box-sizing: border-box;
  border-radius: 14px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  color: var(--text);
  user-select: none;
  overflow: hidden;
  animation: pm-fade-up 0.35s ease both;
}

/* 强调条：3px 竖条紧贴左边缘。使用休息前阶段色，
   让提醒读起来像警告而非响亮图标或动画圆点。 */
.accent {
  width: 3px;
  height: 100%;
  background: var(--phase-prebreak);
  flex-shrink: 0;
}

/* 倒计时簇：数字 + 单位，基线对齐。数字与休息遮罩的细字重美学一致，
   让两个屏幕感觉相关。 */
.countdown {
  display: flex;
  align-items: baseline;
  gap: 3px;
  flex-shrink: 0;
  margin-left: 14px;
  color: var(--phase-prebreak);
}
.num {
  font-size: 34px;
  font-weight: 200;
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

/* 推迟按钮：幽灵文字按钮。无玻璃卡片、无边框 -
   仅一个悬停时变暖的安静交互。与休息遮罩的克制药丸一致。 */
.postpone {
  flex-shrink: 0;
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 500;
  font-family: inherit;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.postpone:hover {
  background: var(--accent-soft);
  color: var(--text);
}
.postpone:active {
  background: var(--accent);
  color: var(--on-accent);
}
</style>
