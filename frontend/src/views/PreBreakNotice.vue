<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { useI18n } from 'vue-i18n'
import { BreakService } from '../../bindings/pocketmind'
import { Button } from '@/components/ui/button'

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
    <div class="notice">
      <span class="dot" aria-hidden="true" />
      <div class="body">
        <div class="title">{{ t('notice.title', { sec: remaining }) }}</div>
        <div class="sub">{{ t('notice.sub') }}</div>
      </div>
      <Button variant="secondary" size="sm" @click="postpone">{{ t('notice.postpone') }}</Button>
    </div>
  </div>
</template>

<style scoped>
.notice-wrap {
  width: 100%;
  height: 100%;
  padding: 8px;
  box-sizing: border-box;
}
.notice {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  height: 100%;
  padding: 0 18px;
  box-sizing: border-box;
  background: var(--card);
  border-radius: 16px;
  color: var(--card-foreground);
  user-select: none;
  border: 1px solid var(--border);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
}
.dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #ffb454;
  box-shadow: 0 0 12px rgba(255, 180, 84, 0.9);
  flex-shrink: 0;
  animation: pulse 1.4s ease-in-out infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
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
  color: var(--muted-foreground);
  margin-top: 2px;
}
</style>
