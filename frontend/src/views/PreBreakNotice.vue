<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import GButton from '@/components/GButton.vue'

const { t } = useI18n()
const remaining = ref(0)
let off: (() => void) | undefined

onMounted(async () => {
  const st = await BreakService.GetState()
  remaining.value = st.remainingSec
  off = Events.On('pm:tick', (ev: { data: { remainingSec: number } }) => {
    remaining.value = ev.data.remainingSec
  })
})
onUnmounted(() => off?.())

function postpone() { BreakService.PostponeBreak() }
</script>

<template>
  <div class="notice-wrap">
    <div class="notice glass-strong">
      <div class="ring" aria-hidden="true">
        <span class="dot" />
      </div>
      <div class="body">
        <div class="title">{{ t('notice.title', { sec: remaining }) }}</div>
        <div class="sub">{{ t('notice.sub') }}</div>
      </div>
      <GButton variant="glass" size="sm" @click="postpone">{{ t('notice.postpone') }}</GButton>
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
.notice {
  display: flex;
  align-items: center;
  gap: 13px;
  width: 100%;
  height: 100%;
  padding: 0 16px;
  box-sizing: border-box;
  border-radius: 14px;
  color: var(--text);
  user-select: none;
  animation: pm-fade-up 0.4s ease both;
}
.ring {
  position: relative;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #ffb454;
  box-shadow: 0 0 10px rgba(255, 180, 84, 0.9);
  animation: pm-breathe 1.6s ease-in-out infinite;
}
.body {
  flex: 1;
  min-width: 0;
}
.title {
  font-size: 14px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.sub {
  font-size: 11.5px;
  color: var(--text-faint);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
