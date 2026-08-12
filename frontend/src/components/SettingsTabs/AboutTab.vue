<script setup lang="ts">
// AboutTab - 关于标签页：应用信息与 20-20-20 规则说明。
import GlassPanel from '@/components/GlassPanel.vue'
import { Sparkles } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
// 从 package.json 读取版本号，避免在 UI 里硬编码导致版本漂移。
import pkg from '../../../package.json'

const { t } = useI18n()
const version = pkg.version
</script>

<template>
  <section v-show="true" class="panel-stack">
    <GlassPanel class="about">
      <div class="about__head">
        <img class="about__logo-img" src="/logo.png" alt="Blink" />
        <div>
          <div class="about__name">{{ t('app.name') }}</div>
          <div class="about__ver">{{ t('about.version') }} {{ version }}</div>
        </div>
      </div>
      <p class="about__desc">{{ t('about.description') }}</p>
      <div class="about__rule">
        <div class="about__rule-title"><Sparkles class="size-4" />{{ t('about.rule') }}</div>
        <p class="about__rule-desc">{{ t('about.ruleDesc') }}</p>
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
.about { padding: 24px; }
.about__head { display: flex; gap: 14px; align-items: center; margin-bottom: 16px; }
.about__logo-img { width: 46px; height: 46px; object-fit: contain; }
.about__name { font-size: 18px; font-weight: 600; }
.about__ver { font-size: 12px; color: var(--text-faint); margin-top: 2px; }
.about__desc { font-size: 13px; color: var(--text-muted); line-height: 1.6; margin: 0 0 16px; }
.about__rule { padding: 16px; border-radius: 12px; background: var(--accent-soft); }
.about__rule-title { font-size: 13px; font-weight: 600; display: flex; align-items: center; gap: 8px; color: var(--accent); margin-bottom: 6px; }
.about__rule-desc { font-size: 13px; color: var(--text-muted); margin: 0; line-height: 1.5; }
</style>
