import { createI18n } from 'vue-i18n'
import { computed } from 'vue'
import { en as uiEnglish, zh_cn as uiChinese } from '@nuxt/ui/locale'
import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'
import errorRules from '../../internal/messages/errors.json'

export const i18n = createI18n({ legacy: false, locale: 'en', fallbackLocale: 'en', messages: { en, 'zh-CN': zhCN } })
export const t = (...args) => i18n.global.t(...args)
export const locale = i18n.global.locale
export const uiLocale = computed(() => locale.value === 'zh-CN' ? uiChinese : uiEnglish)
export const formatLocale = () => locale.value === 'zh-CN' ? 'zh-CN' : 'en-GB'
export const formatNumber = (number, options = {}) => new Intl.NumberFormat(formatLocale(), options).format(number)
let serverDefault = 'en'
export const normaliseLanguage = value => ['en', 'zh-CN'].includes(value) ? value : 'en'
export function setLanguage(value) {
  locale.value = normaliseLanguage(value || serverDefault)
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale.value
    document.title = t('ui.mc3cd485558')
  }
}
export function setServerLanguage(value) {
  serverDefault = normaliseLanguage(value)
  setLanguage('')
}

// Only application-owned messages are translated. Unrecognised external
// diagnostics, printer/device names and document content are returned verbatim.
const errorPatterns = errorRules.map(rule => ({ ...rule, expression: new RegExp(rule.pattern) }))
export function translateError(message, code, params, depth = 0) {
  if (depth > 8) return message || ''
  const nested = (rule, values) => {
    const result = { ...values }
    const slots = rule?.source.match(/%[qdswfv]/g) || []
    slots.forEach((slot, i) => {
      if (['%v', '%w'].includes(slot) && typeof result['p' + i] === 'string') {
        result['p' + i] = translateError(result['p' + i], undefined, undefined, depth + 1)
      }
    })
    return result
  }
  if (code && i18n.global.te(code)) return t(code, nested(errorPatterns.find(rule => rule.code === code), params || {}))
  for (const rule of errorPatterns) {
    const match = rule.expression.exec(message || '')
    if (match) return t(rule.code, nested(rule, Object.fromEntries(match.slice(1).map((value, i) => ['p' + i, value]))))
  }
  return message || ''
}
