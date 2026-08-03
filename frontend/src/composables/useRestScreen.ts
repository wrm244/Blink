// 休息屏幕内容（每日一句 + Bing 壁纸）的获取逻辑。
// 在休息遮罩出现时从 xygeng.cn 开放 API 拉取内容。
// 所有失败都静默回退到纯色背景，确保离线时不比之前更差。
// 日期键按天缓存 Bing 图片（"今日图片"无需每分钟重新获取），
// 每日一句每次休息重新获取以保持新鲜感。

import { ref } from 'vue'

export function useRestScreen() {
  const quote = ref('')
  const quoteMeta = ref('')
  const bgUrl = ref('')
  const dayKey = new Date().toISOString().slice(0, 10)

  /** 并行获取每日一句和 Bing 壁纸 */
  function enrichRestScreen() {
    fetchOne()
    fetchBing()
  }

  /** 从开放 API 获取每日一句 */
  async function fetchOne() {
    try {
      const res = await fetch('https://api.xygeng.cn/openapi/one')
      if (!res.ok) return
      const json = await res.json()
      const content = json?.data?.content
      if (typeof content !== 'string' || !content.trim()) return
      quote.value = content.trim()
      const origin = json.data.origin
      const name = json.data.name
      quoteMeta.value = [origin, name].filter(Boolean).join(' · ')
    } catch {
      /* 离线 - 保持纯色背景 */
    }
  }

  /** 获取 Bing 每日壁纸（按天缓存） */
  async function fetchBing() {
    try {
      const cached = sessionStorage.getItem('pm:bing')
      if (cached) {
        const parsed = JSON.parse(cached)
        if (parsed?.day === dayKey && typeof parsed.url === 'string') {
          bgUrl.value = parsed.url
          return
        }
      }
      const res = await fetch('https://api.xygeng.cn/openapi/bing', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ index: 0 }),
      })
      if (!res.ok) return
      const json = await res.json()
      const urls = json?.data?.urls
      if (!Array.isArray(urls) || urls.length === 0) return
      // urls 按分辨率从高到低排序，urls[0] 是 1920×1080 变体
      const url = urls[0]
      try {
        sessionStorage.setItem('pm:bing', JSON.stringify({ day: dayKey, url }))
      } catch { /* 存储已满 - 缓存是尽力而为的 */ }
      bgUrl.value = url
    } catch {
      /* 离线 - 保持纯色背景 */
    }
  }

  return { quote, quoteMeta, bgUrl, enrichRestScreen }
}
