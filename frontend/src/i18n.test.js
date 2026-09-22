import { describe, it, expect, afterEach } from 'vitest'
import { i18n, t, locale, setLanguage, setServerLanguage, formatLocale, translateError, uiLocale } from './i18n'
import { validPassword } from './utils/account'

afterEach(() => setServerLanguage('en'))
describe('language precedence and formatting', () => {
  it('uses the server default, then an explicit account preference, without browser detection', () => {
    setServerLanguage('zh-CN')
    expect(locale.value).toBe('zh-CN')
    setLanguage('en')
    expect(locale.value).toBe('en')
    expect(formatLocale()).toBe('en-GB')
    expect(document.documentElement.lang).toBe('en')
    expect(uiLocale.value.code).toBe('en')
    setLanguage('')
    expect(locale.value).toBe('zh-CN')
    expect(document.documentElement.lang).toBe('zh-CN')
    expect(t('account.title')).toBe('账户设置')
  })
  it('falls back to English for an invalid default', () => {
    setServerLanguage('invalid')
    expect(locale.value).toBe('en')
    expect(i18n.global.fallbackLocale.value).toBe('en')
  })
  it('translates application errors and retains external diagnostics verbatim', () => {
    setLanguage('en')
    expect(translateError('缺少 device')).toBe('Missing device')
    expect(translateError('scanimage 失败: external printer diagnostic')).toBe('scanimage failed: external printer diagnostic')
    expect(translateError('列出扫描设备失败: scanimage -L 失败: external diagnostic')).toBe('Could not list scanners: scanimage -L failed: external diagnostic')
    expect(translateError('外部打印机原始错误 0x123')).toBe('外部打印机原始错误 0x123')
    expect(translateError('', 'account.current_password_wrong')).toBe('The current password is incorrect')
  })
})
describe('password limits', () => {
  it('counts characters and UTF-8 bytes separately', () => {
    expect(validPassword('1234567')).toBe(false)
    expect(validPassword('12345678')).toBe(true)
    expect(validPassword('a'.repeat(72))).toBe(true)
    expect(validPassword('a'.repeat(73))).toBe(false)
    expect(validPassword('字'.repeat(24))).toBe(true)
    expect(validPassword('字'.repeat(25))).toBe(false)
    expect(validPassword('😀'.repeat(8))).toBe(true)
  })
})
