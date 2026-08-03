<script setup lang="ts">
// CloseConfirmDialog - 关闭设置窗口时的确认弹窗。
//
// 由 Go 端在拦截窗口关闭后通过 blink:close-request 事件唤起（目前仅 Windows）。
// 之所以自绘而不用系统对话框：Windows 的 MessageBox 无法自定义按钮文字，
// 也没有"记住我的选择"复选框，详见 prefs_close_windows.go 的说明。
import GButton from '@/components/GButton.vue'
import { Check, LogOut, MinusCircle } from '@lucide/vue'
import { onMounted, onUnmounted, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

// 与 Go 端约定的动作字符串：background / quit 直接对应 Settings.CloseAction
// 的取值，cancel 表示用户放弃关闭。
type CloseChoice = 'background' | 'quit' | 'cancel'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ resolve: [action: CloseChoice, remember: boolean] }>()

const { t } = useI18n()
const remember = ref(false)
const primaryBtn = ref<InstanceType<typeof GButton> | null>(null)

// 每次重新唤起都复位勾选并把焦点放到主按钮上：
// 上次的勾选状态若残留，用户可能在没注意的情况下把选择固化下来。
watch(() => props.open, async (v) => {
  if (!v) return
  remember.value = false
  await nextTick()
  ;(primaryBtn.value?.$el as HTMLElement | undefined)?.focus()
})

function choose(action: 'background' | 'quit') { emit('resolve', action, remember.value) }
function cancel() { emit('resolve', 'cancel', false) }

// ESC 取消。监听器常驻组件生命周期，靠 props.open 判断是否响应，
// 这样弹窗的进出场动画不会受监听器装卸的影响。
function onKeydown(e: KeyboardEvent) {
  if (props.open && e.key === 'Escape') {
    e.preventDefault()
    cancel()
  }
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Transition name="pm-dialog">
    <div v-if="open" class="mask" @click.self="cancel">
      <div class="dlg" role="dialog" aria-modal="true">
        <h2 class="dlg__title">{{ t('closeDialog.title') }}</h2>
        <p class="dlg__question">{{ t('closeDialog.question') }}</p>

        <ul class="dlg__hints">
          <li><MinusCircle class="size-3.5 shrink-0" /><span>{{ t('closeDialog.bgHint') }}</span></li>
          <li><LogOut class="size-3.5 shrink-0" /><span>{{ t('closeDialog.quitHint') }}</span></li>
        </ul>

        <button
          class="ck"
          type="button"
          role="checkbox"
          :aria-checked="remember"
          @click="remember = !remember"
        >
          <span class="ck__box" :class="{ 'ck__box--on': remember }">
            <Check v-if="remember" class="size-3" />
          </span>
          <span class="ck__text">
            {{ t('closeDialog.remember') }}
            <span class="ck__hint">{{ t('closeDialog.rememberHint') }}</span>
          </span>
        </button>

        <div class="dlg__actions">
          <GButton variant="ghost" size="sm" @click="cancel">{{ t('closeDialog.cancel') }}</GButton>
          <div class="dlg__spacer" />
          <GButton variant="danger" size="sm" @click="choose('quit')">
            <LogOut class="size-3.5" />{{ t('closeDialog.quit') }}
          </GButton>
          <GButton ref="primaryBtn" variant="primary" size="sm" @click="choose('background')">
            {{ t('closeDialog.background') }}
          </GButton>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed; inset: 0; z-index: 50;
  display: grid; place-items: center;
  padding: 24px;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(10px);
}

/* 弹窗用不透明实色背景，不走 --glass-* 变量：
   暗色主题下 --glass-bg-strong 仅 7% 不透明度，模态框叠在半透明遮罩上
   会让文字直接透出后面的应用内容，根本看不清。 */
.dlg {
  --dlg-bg: #ffffff;
  --dlg-bg-subtle: #f4f5f7;
  --dlg-border: rgba(0, 0, 0, 0.08);
  --dlg-ck-border: rgba(0, 0, 0, 0.2);
  width: 100%; max-width: 420px;
  padding: 22px 24px 18px;
  border-radius: 16px;
  background: var(--dlg-bg);
  border: 1px solid var(--dlg-border);
  box-shadow: 0 24px 60px -12px rgba(0, 0, 0, 0.6);
}
.dark .dlg {
  --dlg-bg: #262932;
  --dlg-bg-subtle: #2d303a;
  --dlg-border: rgba(255, 255, 255, 0.1);
  --dlg-ck-border: rgba(255, 255, 255, 0.15);
}
.dlg__title { margin: 0; font-size: 16px; font-weight: 600; letter-spacing: -0.01em; color: var(--text); }
.dlg__question { margin: 8px 0 0; font-size: 13.5px; line-height: 1.5; color: var(--text-muted); }

.dlg__hints { margin: 14px 0 0; padding: 12px 14px; list-style: none; display: flex; flex-direction: column; gap: 9px; border-radius: 11px; background: var(--dlg-bg-subtle); border: 1px solid var(--dlg-border); }
.dlg__hints li { display: flex; align-items: flex-start; gap: 8px; font-size: 12.5px; line-height: 1.5; color: var(--text-muted); }
.dlg__hints svg { margin-top: 2px; color: var(--text-faint); }

/* 记住选择 */
.ck { display: flex; align-items: flex-start; gap: 9px; width: 100%; margin-top: 14px; padding: 8px; font-family: inherit; text-align: left; background: transparent; border: 1px solid transparent; border-radius: 9px; cursor: pointer; transition: background 0.15s; }
.ck:hover { background: var(--accent-soft); }
.ck:focus-visible { outline: none; border-color: var(--accent); }
.ck__box { display: grid; place-items: center; width: 16px; height: 16px; margin-top: 1px; flex-shrink: 0; border-radius: 5px; border: 1px solid var(--dlg-ck-border); background: var(--track); color: var(--on-accent); transition: background 0.15s, border-color 0.15s; }
.ck__box--on { background: var(--accent); border-color: var(--accent); }
.ck__text { display: flex; flex-direction: column; gap: 2px; font-size: 12.5px; color: var(--text); }
.ck__hint { font-size: 11px; color: var(--text-faint); }

.dlg__actions { display: flex; align-items: center; gap: 8px; margin-top: 16px; }
.dlg__spacer { flex: 1; }

.pm-dialog-enter-active, .pm-dialog-leave-active { transition: opacity 0.18s ease; }
.pm-dialog-enter-active .dlg, .pm-dialog-leave-active .dlg { transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.2s; }
.pm-dialog-enter-from, .pm-dialog-leave-to { opacity: 0; }
.pm-dialog-enter-from .dlg, .pm-dialog-leave-to .dlg { opacity: 0; transform: translateY(10px) scale(0.97); }
</style>
