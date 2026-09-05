<script setup lang="ts">
// OptionsTab - 选项标签页：分组列表风格（与时间页/关于页同构）。
// 通用 / 自动暂停 / 窗口（仅 Windows）三组；行为开关用 GToggle，
// 关闭行为用 GSegmented。行排版与 GTimeField 一致（13.5px 标签 + 11.5px 描述）。
import GlassPanel from '@/components/GlassPanel.vue'
import GToggle from '@/components/GToggle.vue'
import GSegmented from '@/components/GSegmented.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isWindows } from '@/lib/platform'
import type { Settings } from '@bindings/blink/internal/config/models'

const { t } = useI18n()

defineProps<{ s: Settings }>()
const emit = defineEmits<{
  'update': [key: keyof Settings, value: boolean | string]
}>()

// 仅 Windows 需要这个开关：macOS 关窗天然回到菜单栏，不存在"要不要留后台"的问题。
const showCloseAction = isWindows

// 必须是 computed —— 语言是运行时切换的（i18n.global.locale.value），
// 普通数组会把 t() 的结果固化在组件创建那一刻，切语言后标签不会更新。
const closeOptions = computed(() => [
  { value: 'ask', label: t('options.closeAsk') },
  { value: 'background', label: t('options.closeBackground') },
  { value: 'quit', label: t('options.closeQuit') },
])
</script>

<template>
  <section v-show="true" class="panel-stack">
    <!-- 通用 -->
    <GlassPanel class="grp">
      <div class="grp__title">{{ t('options.groupGeneral') }}</div>
      <div class="row">
        <div class="row__text">
          <div class="row__label">{{ t('options.autoStart') }}</div>
          <div class="row__desc">{{ t('options.autoStartDesc') }}</div>
        </div>
        <GToggle :model-value="s.autoStart" @update:model-value="(v: boolean) => emit('update', 'autoStart', v)" />
      </div>
      <div class="row">
        <div class="row__text">
          <div class="row__label">{{ t('options.launchAtLogin') }}</div>
          <div class="row__desc">{{ t('options.launchAtLoginDesc') }}</div>
        </div>
        <GToggle :model-value="s.launchAtLogin" @update:model-value="(v: boolean) => emit('update', 'launchAtLogin', v)" />
      </div>
      <div class="row">
        <div class="row__text">
          <div class="row__label">{{ t('options.sound') }}</div>
          <div class="row__desc">{{ t('options.soundDesc') }}</div>
        </div>
        <GToggle :model-value="s.soundEnabled" @update:model-value="(v: boolean) => emit('update', 'soundEnabled', v)" />
      </div>
    </GlassPanel>

    <!-- 自动暂停 -->
    <GlassPanel class="grp">
      <div class="grp__title">{{ t('options.groupAutoPause') }}</div>
      <div class="row">
        <div class="row__text">
          <div class="row__label">{{ t('options.pauseOnMeeting') }}</div>
          <div class="row__desc">{{ t('options.pauseOnMeetingDesc') }}</div>
        </div>
        <GToggle :model-value="s.pauseOnMeeting" @update:model-value="(v: boolean) => emit('update', 'pauseOnMeeting', v)" />
      </div>
      <div class="row">
        <div class="row__text">
          <div class="row__label">{{ t('options.pauseOnMedia') }}</div>
          <div class="row__desc">{{ t('options.pauseOnMediaDesc') }}</div>
        </div>
        <GToggle :model-value="s.pauseOnMedia" @update:model-value="(v: boolean) => emit('update', 'pauseOnMedia', v)" />
      </div>
    </GlassPanel>

    <!-- 窗口（仅 Windows） -->
    <GlassPanel v-if="showCloseAction" class="grp">
      <div class="grp__title">{{ t('options.groupWindow') }}</div>
      <div class="row row--wrap">
        <div class="row__text">
          <div class="row__label">{{ t('options.closeTitle') }}</div>
          <div class="row__desc">{{ t('options.closeDesc') }}</div>
        </div>
        <GSegmented
          :options="closeOptions"
          :model-value="s.closeAction || 'ask'"
          @update:model-value="(v: string) => emit('update', 'closeAction', v)"
        />
      </div>
    </GlassPanel>
  </section>
</template>

<style scoped>
.panel-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: pm-fade-up 0.4s ease both;
}
/* 分组面板与标题：与 TimingTab / AboutTab 完全一致 */
.grp { padding: 16px 0 4px; }
.grp__title {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
  padding: 0 18px 12px;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 18px;
}
.row + .row { border-top: 1px solid var(--glass-border); }
/* GSegmented 选项较宽，窄窗口时允许换行避免溢出 */
.row--wrap { flex-wrap: wrap; }
.row__text { min-width: 0; }
.row__label { font-size: 13.5px; font-weight: 600; color: var(--text); }
.row__desc { font-size: 11.5px; color: var(--text-faint); margin-top: 2px; line-height: 1.45; }
</style>
