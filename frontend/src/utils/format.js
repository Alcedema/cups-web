
import { t, formatLocale, formatNumber } from '../i18n.js'
export function formatFileSize(bytes) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return formatNumber(bytes / 1024, { maximumFractionDigits: 1 }) + ' KB'
  return formatNumber(bytes / (1024 * 1024), { maximumFractionDigits: 1 }) + ' MB'
}

export function formatTime(iso) {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleString(formatLocale(), { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  } catch { return iso }
}

export function formatPrinterName(uri) {
  if (!uri) return ''
  const parts = uri.split('/')
  return parts[parts.length - 1] || uri
}

// printerLabel 拼打印机下拉的主标签：队列名 + CUPS 描述（issue #101）。
// CUPS 建队列时若没指定 -D，printer-info 会默认等于队列名，这时不重复显示。
export function printerLabel(printer) {
  if (!printer) return ''
  const name = printer.name || formatPrinterName(printer.uri)
  const info = (printer.info || '').trim()
  if (info && info.toLowerCase() !== name.toLowerCase()) return `${name} — ${info}`
  return name
}

// printerDescription 拼下拉项的副标题：型号 · 位置 · URI。
// 描述也重名时靠这一行区分（同型号多台、或多台机器共用一个描述）。
export function printerDescription(printer) {
  if (!printer) return ''
  return [
    (printer.makeAndModel || '').trim(),
    (printer.location || '').trim(),
    printer.uri
  ].filter(Boolean).join(' · ')
}

export function formatDurationSeconds(totalSeconds) {
  if (!totalSeconds || totalSeconds < 0) return t('ui.m4d8c1c5b42')
  const d = Math.floor(totalSeconds / 86400)
  const h = Math.floor((totalSeconds % 86400) / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  if (d > 0) return t('ui.m625fcdb686', { p0: (d), p1: (h) })
  if (h > 0) return t('ui.m6b4da5dfa4', { p0: (h), p1: (m) })
  if (m > 0) return t('ui.me235256e97', { p0: (m) })
  return t('ui.me9dc5dd213', { p0: (totalSeconds) })
}

export function statusColor(status) {
  const map = { queued: 'info', printed: 'success', failed: 'error', cancelled: 'neutral' }
  return map[status] || 'neutral'
}

export function statusText(status) {
  const map = { queued: t('ui.md6f766f2ad'), printed: t('ui.m59808ada86'), failed: t('ui.m28384d7afd'), cancelled: t('ui.ma37778f17c') }
  return map[status] || status
}

export function printerStateColor(state) {
  const map = { idle: 'success', processing: 'warning', stopped: 'error' }
  return map[state] || 'neutral'
}

export function printerStateText(state) {
  const map = { idle: t('ui.mdae661d17c'), processing: t('ui.m2978542fbb'), stopped: t('ui.mf006455e3b') }
  return map[state] || state || t('ui.m4d8c1c5b42')
}

export function markerLevelColor(level) {
  if (level === undefined || level === null) return 'text-muted'
  if (level <= 10) return 'text-error font-bold'
  if (level <= 25) return 'text-warning font-medium'
  return 'text-success'
}

export function markerBarColor(level) {
  if (level === undefined || level === null) return 'bg-muted'
  if (level <= 10) return 'bg-error'
  if (level <= 25) return 'bg-warning'
  return 'bg-success'
}
