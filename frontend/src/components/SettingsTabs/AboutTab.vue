<script setup lang="ts">
// AboutTab - 关于标签页：分组列表风格（与时间页同构），纯文本行。
// 应用信息 / 20-20-20 规则 / 开源信息 三组；仓库行可点击，
// 其余为静态行。无图标、无 logo——设置页 hero 已有品牌区，这里保持克制。
import GlassPanel from '@/components/GlassPanel.vue'
import { ArrowUpRight } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
// 从 package.json 读取版本号，避免在 UI 里硬编码导致版本漂移。
import pkg from '../../../package.json'

const { t } = useI18n()
const version = pkg.version
const repoUrl = 'https://github.com/wrm244/Blink'

function openRepo() {
  window.open(repoUrl, '_blank')
}
</script>

<template>
  <section v-show="true" class="panel-stack">
    <!-- 应用 -->
    <GlassPanel class="grp">
      <div class="grp__title">{{ t('about.intro') }}</div>
      <div class="row">
        <div class="row__text">
          <div class="row__label">
            {{ t('app.name') }}
            <span class="row__ver">v{{ version }}</span>
          </div>
          <div class="row__desc">{{ t('about.description') }}</div>
        </div>
      </div>
    </GlassPanel>

    <!-- 20-20-20 规则 -->
    <GlassPanel class="grp">
      <div class="grp__title">{{ t('about.rule') }}</div>
      <div class="row">
        <div class="row__text">
          <div class="row__desc">{{ t('about.ruleDesc') }}</div>
        </div>
      </div>
    </GlassPanel>

    <!-- 开源 -->
    <GlassPanel class="grp">
      <div class="grp__title">{{ t('about.openSource') }}</div>
      <button class="row row--link" @click="openRepo">
        <div class="row__text">
          <div class="row__label">{{ t('about.repo') }}</div>
          <div class="row__desc">{{ t('about.repoDesc') }}</div>
        </div>
        <span class="row__value">wrm244/Blink <ArrowUpRight class="size-3.5" /></span>
      </button>
      <div class="row">
        <div class="row__text">
          <div class="row__label">{{ t('about.license') }}</div>
          <div class="row__desc">{{ t('about.licenseDesc') }}</div>
        </div>
        <span class="row__value">MIT</span>
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

/* 分组面板：标题缩进与时间页 .presets / .timing-list 一致（18px） */
.grp { padding: 16px 0 4px; }
.grp__title {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-faint);
  padding: 0 18px 12px;
}

/* 通用行：标签/描述（左），值（右）——与 GTimeField 行同构，纯文本 */
.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 18px;
  width: 100%;
  text-align: left;
  font-family: inherit;
  background: transparent;
  border: none;
  color: var(--text);
}
.row + .row { border-top: 1px solid var(--glass-border); }

.row__text { flex: 1; min-width: 0; }
.row__label {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.row__ver {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-faint);
}
.row__desc {
  font-size: 11.5px;
  color: var(--text-faint);
  margin-top: 2px;
  line-height: 1.45;
}
.row__value {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
  flex-shrink: 0;
}

/* 链接行：hover 高亮，与 nav__item 的交互一致 */
.row--link { cursor: pointer; transition: background 0.15s; }
.row--link:hover { background: var(--accent-soft); }
.row--link:hover .row__value { color: var(--accent); }
</style>
