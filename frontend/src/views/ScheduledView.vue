<template>
  <div class="p-3 sm:p-4 md:p-6 max-w-5xl mx-auto space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <h1 class="text-lg font-bold flex items-center gap-2">
        <UIcon name="i-lucide-calendar-clock" class="w-5 h-5 text-primary" />
        {{ t('ui.m97a57f1b7f') }}
      </h1>
      <div class="flex items-center gap-2">
        <UButton
          variant="ghost"
          size="xs"
          icon="i-lucide-refresh-cw"
          :loading="loading"
          @click="loadRecords"
        >{{ t('ui.maee8874341') }}</UButton>
        <UButton
          size="sm"
          color="primary"
          icon="i-lucide-plus"
          @click="openCreate"
        >{{ t('ui.mf36aa4aa87') }}</UButton>
      </div>
    </div>

    <UAlert
      color="neutral"
      variant="soft"
      icon="i-lucide-info"
      :title="t('ui.mcb4ba40bf8')"
      :description="t('ui.m45f386c6f8')"
    />

    <div v-if="loading" class="flex items-center justify-center py-10 text-muted gap-2">
      <UIcon name="i-lucide-loader-circle" class="w-5 h-5 animate-spin" />
      {{ t('ui.m4927a53bcc') }}
    </div>
    <div v-else-if="!records.length" class="text-center py-16 text-muted">
      <UIcon name="i-lucide-calendar-off" class="w-10 h-10 mx-auto mb-2 opacity-40" />
      <div>{{ t('ui.m6b8e198cee') }}</div>
    </div>
    <div v-else class="space-y-3">
      <UCard v-for="rec in records" :key="rec.id" class="overflow-hidden">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div class="min-w-0 space-y-1">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-medium truncate">{{ rec.name || rec.filename }}</span>
              <UBadge
                :color="rec.enabled ? 'success' : 'neutral'"
                :variant="rec.enabled ? 'subtle' : 'outline'"
                size="sm"
              >{{ rec.enabled ? t('ui.mf4f0ead111') : t('ui.ma8c3698b5b') }}</UBadge>
              <UBadge color="primary" variant="outline" size="sm">
                {{ scheduleLabel(rec) }}
              </UBadge>
              <UBadge
                v-if="rec.lastStatus"
                :color="lastStatusColor(rec.lastStatus)"
                variant="soft"
                size="sm"
              >{{ lastStatusLabel(rec.lastStatus) }}</UBadge>
            </div>
            <div class="text-xs text-muted flex flex-wrap gap-x-4 gap-y-0.5">
              <span>{{ t('ui.m9491ade278') }}{{ rec.filename }}</span>
              <span>{{ t('ui.m30de482794') }}{{ shortPrinter(rec.printerUri) }}</span>
              <span>{{ t('ui.me5227f62d8') }}{{ formatTime(rec.nextRunAt) }}</span>
              <span v-if="rec.lastRunAt">{{ t('ui.m4ce1601064') }}{{ formatTime(rec.lastRunAt) }}</span>
            </div>
            <div v-if="rec.lastError" class="text-xs text-error truncate">
              {{ t('ui.ma4fa181ec3') }}{{ translateError(rec.lastError) }}
            </div>
          </div>
          <div class="flex items-center gap-1 flex-shrink-0">
            <UButton size="xs" variant="ghost" icon="i-lucide-play" :loading="rec._running" @click="runNow(rec)">{{ t('ui.m6912269df2') }}</UButton>
            <UButton size="xs" variant="ghost" :icon="rec.enabled ? 'i-lucide-pause' : 'i-lucide-play-circle'" @click="toggleEnabled(rec)">
              {{ rec.enabled ? t('ui.m4e6fd0e28c') : t('ui.mf4f0ead111') }}
            </UButton>
            <UButton size="xs" variant="ghost" icon="i-lucide-pencil" @click="openEdit(rec)">{{ t('ui.m0518365699') }}</UButton>
            <UButton size="xs" variant="ghost" color="error" icon="i-lucide-trash-2" @click="confirmDelete(rec)">{{ t('ui.m2f9daa8289') }}</UButton>
          </div>
        </div>
      </UCard>
    </div>

    <!-- 新建 / 编辑抽屉 -->
    <UModal v-model:open="modalOpen" :title="editing ? t('ui.m6d931444b2') : t('ui.mf36aa4aa87')" :ui="{ content: 'sm:max-w-2xl' }">
      <template #body>
        <div class="space-y-3">
          <UFormField :label="t('ui.m1cad86800f')">
            <UInput v-model="form.name" :placeholder="t('ui.me5f12e926b')" />
          </UFormField>

          <UFormField v-if="!editing" :label="t('ui.m39932f24fe')" required>
            <input
              type="file"
              class="block w-full text-sm text-default file:mr-3 file:py-1.5 file:px-3 file:rounded-md file:border-0 file:bg-primary/10 file:text-primary hover:file:bg-primary/20"
              @change="onFilePick"
            />
            <div v-if="form.file" class="text-xs text-muted mt-1 truncate">
              {{ t('ui.md4102e5d06') }}{{ form.file.name }}（{{ formatSize(form.file.size) }}）
            </div>
          </UFormField>
          <UFormField v-else :label="t('ui.m39932f24fe')">
            <div class="text-sm text-muted truncate">{{ form.filename }}{{ t('ui.m12e9c26541') }}</div>
          </UFormField>

          <UFormField :label="t('ui.m7d6376ef9f')" required>
            <USelect
              v-model="form.printerUri"
              :items="printerItems"
              value-key="value"
              label-key="label"
              :placeholder="t('ui.md97891fc1e')"
              icon="i-lucide-printer"
            />
          </UFormField>

          <UFormField :label="t('ui.m3c7b79b734')" required>
            <USelect
              v-model="form.scheduleType"
              :items="scheduleTypeItems"
              value-key="value"
              label-key="label"
            />
          </UFormField>

          <UFormField v-if="form.scheduleType === 'once'" :label="t('ui.m45e03e436e')" required>
            <UInput v-model="form.runAtLocal" type="datetime-local" />
          </UFormField>
          <template v-else>
            <UFormField :label="t('ui.m227ebc503f')" required>
              <UInput v-model="form.scheduleTime" type="time" />
            </UFormField>
            <UFormField v-if="form.scheduleType === 'weekly'" :label="t('ui.mc398134d7d')" required>
              <USelect
                v-model.number="form.scheduleWeekday"
                :items="weekdayItems"
                value-key="value"
                label-key="label"
              />
            </UFormField>
            <UFormField v-if="form.scheduleType === 'monthly'" :label="t('ui.m70d0c1b336')" required>
              <UInput v-model.number="form.scheduleDay" type="number" min="1" max="31" />
              <template #hint>{{ t('ui.m28f1be0c2f') }}</template>
            </UFormField>
          </template>

          <UFormField :label="t('ui.mdb9aa2283c')">
            <UInput v-model.number="form.copies" type="number" min="1" max="99" />
          </UFormField>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <UFormField :label="t('ui.m8362011f8e')">
              <USelect v-model="form.paperSize" :items="paperSizeItems" value-key="value" label-key="label" />
            </UFormField>
            <UFormField :label="t('ui.m1121471a0f')">
              <USelect v-model="form.orientation" :items="orientationItems" value-key="value" label-key="label" />
            </UFormField>
            <UFormField :label="t('ui.med80ebb1cb')">
              <USelect v-model="form.duplex" :items="duplexItems" value-key="value" label-key="label" />
            </UFormField>
            <UFormField :label="t('ui.me40fffdb97')">
              <USelect v-model="form.color" :items="colorItems" value-key="value" label-key="label" />
            </UFormField>
          </div>

          <UFormField :label="t('ui.ma905af6ea7')">
            <UInput v-model="form.watermarkText" :placeholder="t('ui.mc6d415b069')" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" @click="modalOpen = false">{{ t('ui.m2cd0f3be87') }}</UButton>
          <UButton color="primary" :loading="submitting" @click="submit">
            {{ editing ? t('ui.ma3030bf8f1') : t('ui.mcde2cd071d') }}
          </UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { t, translateError, formatLocale, formatNumber } from '../i18n.js'

import { ref, computed, onMounted } from 'vue'
import { apiFetch, readError } from '../utils/api'

const emit = defineEmits(['logout'])
const toast = useToast()

const loading = ref(false)
const records = ref([])
const printers = ref([])

const modalOpen = ref(false)
const editing = ref(null)
const submitting = ref(false)

const form = ref(emptyForm())

function emptyForm() {
  return {
    id: null,
    name: '',
    file: null,
    filename: '',
    printerUri: '',
    scheduleType: 'once',
    runAtLocal: defaultRunAtLocal(),
    scheduleTime: '09:00',
    scheduleWeekday: 1,
    scheduleDay: 1,
    copies: 1,
    duplex: 'one-sided',
    color: 'color',
    paperSize: 'A4',
    orientation: 'portrait',
    watermarkText: ''
  }
}

// 默认新任务时间：明天上午 9 点，避免用户不改时间就直接创建导致立即触发。
function defaultRunAtLocal() {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(9, 0, 0, 0)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const scheduleTypeItems = [
  { value: 'once', get label() { return t('ui.m4687f0397f') } },
  { value: 'daily', get label() { return t('ui.meea1694c23') } },
  { value: 'weekly', get label() { return t('ui.m92845d3b53') } },
  { value: 'monthly', get label() { return t('ui.m68b21af949') } }
]

const weekdayItems = [
  { value: 0, get label() { return t('ui.mee239f3943') } },
  { value: 1, get label() { return t('ui.mc430f4c12e') } },
  { value: 2, get label() { return t('ui.md1378a68e6') } },
  { value: 3, get label() { return t('ui.m2961168962') } },
  { value: 4, get label() { return t('ui.me9d3b01a5a') } },
  { value: 5, get label() { return t('ui.m572388f3b0') } },
  { value: 6, get label() { return t('ui.mf9aa11dbb1') } }
]

const paperSizeItems = [
  { value: 'A4', label: 'A4' },
  { value: 'A3', label: 'A3' },
  { value: 'Letter', label: 'Letter' },
  { value: '5inch', get label() { return t('ui.med4e894462') } },
  { value: '6inch', get label() { return t('ui.mba0a18ab5a') } },
  { value: '7inch', get label() { return t('ui.m260b01cb83') } },
  { value: '8inch', get label() { return t('ui.m548a1f94dc') } },
  { value: '10inch', get label() { return t('ui.m52f987f7e9') } }
]
const orientationItems = [
  { value: 'portrait', get label() { return t('ui.m8d48cd5dd4') } },
  { value: 'landscape', get label() { return t('ui.md95352f4e0') } }
]
const duplexItems = [
  { value: 'one-sided', get label() { return t('ui.mdf608c3e7d') } },
  { value: 'two-sided-long-edge', get label() { return t('ui.ma2a3e273d2') } }
]
const colorItems = [
  { value: 'color', get label() { return t('ui.mdb57813f39') } },
  { value: 'monochrome', get label() { return t('ui.m47d88357c5') } }
]

const printerItems = computed(() =>
  printers.value.map((p) => ({ value: p.uri || p.deviceUri || p.name, label: p.name || p.uri }))
)

function shortPrinter(uri) {
  if (!uri) return ''
  const idx = uri.lastIndexOf('/')
  return idx >= 0 ? uri.slice(idx + 1) : uri
}

function formatTime(iso) {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString(formatLocale())
  } catch (e) {
    return iso
  }
}

function formatSize(bytes) {
  if (!bytes) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${formatNumber(n, { minimumFractionDigits: 1, maximumFractionDigits: 1 })} ${units[i]}`
}

function scheduleLabel(rec) {
  switch (rec.scheduleType) {
    case 'once': return t('ui.m4687f0397f')
    case 'daily': return t('ui.m1cd663150c', { p0: (rec.scheduleTime) })
    case 'weekly': return t('ui.mbe4f700f2f', { p0: (weekdayLabel(rec.scheduleWeekday)), p1: (rec.scheduleTime) })
    case 'monthly': return t('ui.m507f878999', { p0: (rec.scheduleDay), p1: (rec.scheduleTime) })
    default: return rec.scheduleType
  }
}
function weekdayLabel(w) {
  return [t('ui.mee239f3943'), t('ui.mc430f4c12e'), t('ui.md1378a68e6'), t('ui.m2961168962'), t('ui.me9d3b01a5a'), t('ui.m572388f3b0'), t('ui.mf9aa11dbb1')][((w % 7) + 7) % 7]
}
function lastStatusLabel(s) {
  if (s === 'printed') return t('ui.m796d96bcae')
  if (s === 'failed') return t('ui.mba26774ea8')
  if (s === 'skipped') return t('ui.mc4b4626007')
  return s
}
function lastStatusColor(s) {
  if (s === 'printed') return 'success'
  if (s === 'failed') return 'error'
  if (s === 'skipped') return 'warning'
  return 'neutral'
}

async function loadRecords() {
  loading.value = true
  try {
    const resp = await apiFetch('/api/scheduled-prints', {}, () => emit('logout'))
    if (!resp.ok) throw new Error(await readError(resp))
    records.value = await resp.json()
  } catch (e) {
    toast.add({ title: t('ui.md1d044826a'), description: translateError(e.message), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    loading.value = false
  }
}

async function loadPrinters() {
  try {
    const resp = await apiFetch('/api/printers', {}, () => emit('logout'))
    if (!resp.ok) return
    printers.value = await resp.json()
  } catch (e) {
    // 静默失败：定时任务表单没有打印机数据时会给出必填提示
  }
}

function openCreate() {
  editing.value = null
  form.value = emptyForm()
  if (printerItems.value.length) {
    form.value.printerUri = printerItems.value[0].value
  }
  modalOpen.value = true
}

function openEdit(rec) {
  editing.value = rec
  form.value = {
    id: rec.id,
    name: rec.name || '',
    file: null,
    filename: rec.filename,
    printerUri: rec.printerUri,
    scheduleType: rec.scheduleType,
    runAtLocal: rec.runAt ? toLocalInput(rec.runAt) : defaultRunAtLocal(),
    scheduleTime: rec.scheduleTime || '09:00',
    scheduleWeekday: rec.scheduleWeekday,
    scheduleDay: rec.scheduleDay || 1,
    copies: rec.copies || 1,
    duplex: rec.isDuplex ? 'two-sided-long-edge' : 'one-sided',
    color: rec.isColor ? 'color' : 'monochrome',
    paperSize: rec.paperSize || 'A4',
    orientation: rec.orientation || 'portrait',
    watermarkText: rec.watermarkText || ''
  }
  modalOpen.value = true
}

// toLocalInput 把服务器返回的 UTC RFC3339 转成 <input type="datetime-local"> 需要的
// 本地时区 YYYY-MM-DDTHH:mm 字符串。用户在本地看到的日期时间与实际触发对齐。
function toLocalInput(iso) {
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function onFilePick(e) {
  const files = e.target?.files
  form.value.file = files && files.length ? files[0] : null
}

async function submit() {
  const v = form.value
  if (!v.printerUri) { toast.add({ title: t('ui.me0308fd243'), color: 'warning' }); return }
  if (!editing.value && !v.file) { toast.add({ title: t('ui.m6c9e0d9710'), color: 'warning' }); return }
  if (v.scheduleType === 'once' && !v.runAtLocal) { toast.add({ title: t('ui.m434c7efecb'), color: 'warning' }); return }

  submitting.value = true
  try {
    if (editing.value) {
      await saveEdit(v)
    } else {
      await createNew(v)
    }
    modalOpen.value = false
    await loadRecords()
  } catch (e) {
    toast.add({ title: t('ui.m6309a3bb5b'), description: translateError(e.message), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    submitting.value = false
  }
}

async function createNew(v) {
  const fd = new FormData()
  fd.append('file', v.file, v.file.name)
  fd.append('name', v.name)
  fd.append('printer', v.printerUri)
  fd.append('copies', String(v.copies))
  fd.append('duplex', v.duplex === 'two-sided-long-edge' ? 'true' : 'false')
  fd.append('color', v.color === 'color' ? 'true' : 'false')
  fd.append('orientation', v.orientation)
  fd.append('paper_size', v.paperSize)
  fd.append('paper_type', 'plain')
  fd.append('media_source', 'auto')
  fd.append('print_scaling', 'fit')
  fd.append('page_set', 'all')
  fd.append('number_up', '1')
  fd.append('number_up_layout', 'lrtb')
  fd.append('page_border', 'none')
  if (v.watermarkText) fd.append('watermark_text', v.watermarkText)
  fd.append('schedule_type', v.scheduleType)
  if (v.scheduleType === 'once') {
    fd.append('run_at', new Date(v.runAtLocal).toISOString())
  } else {
    fd.append('schedule_time', v.scheduleTime)
    fd.append('schedule_weekday', String(v.scheduleWeekday))
    fd.append('schedule_day', String(v.scheduleDay))
  }
  const resp = await apiFetch('/api/scheduled-prints', { method: 'POST', body: fd }, () => emit('logout'))
  if (!resp.ok) throw new Error(await readError(resp))
  toast.add({ title: t('ui.m6072bcf9f2'), color: 'success', icon: 'i-lucide-check-circle' })
}

async function saveEdit(v) {
  const payload = {
    name: v.name,
    printerUri: v.printerUri,
    isDuplex: v.duplex === 'two-sided-long-edge',
    isColor: v.color === 'color',
    copies: v.copies,
    orientation: v.orientation,
    paperSize: v.paperSize,
    paperType: 'plain',
    mediaSource: 'auto',
    printScaling: 'fit',
    pageRange: '',
    pageSet: 'all',
    mirror: false,
    watermarkText: v.watermarkText,
    numberUp: 1,
    numberUpLayout: 'lrtb',
    pageBorder: 'none',
    scheduleType: v.scheduleType,
    scheduleTime: v.scheduleTime,
    scheduleWeekday: v.scheduleWeekday,
    scheduleDay: v.scheduleDay,
    runAt: v.scheduleType === 'once' ? new Date(v.runAtLocal).toISOString() : ''
  }
  const resp = await apiFetch(`/api/scheduled-prints/${v.id}`, {
    method: 'PUT',
    body: JSON.stringify(payload)
  }, () => emit('logout'))
  if (!resp.ok) throw new Error(await readError(resp))
  toast.add({ title: t('ui.m434203a723'), color: 'success', icon: 'i-lucide-check-circle' })
}

async function runNow(rec) {
  rec._running = true
  try {
    const resp = await apiFetch(`/api/scheduled-prints/${rec.id}/run`, { method: 'POST' }, () => emit('logout'))
    if (!resp.ok) throw new Error(await readError(resp))
    toast.add({ title: t('ui.md4dcb5d349'), color: 'success', icon: 'i-lucide-check-circle' })
    await loadRecords()
  } catch (e) {
    toast.add({ title: t('ui.m8c7f1a15b3'), description: translateError(e.message), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    rec._running = false
  }
}

async function toggleEnabled(rec) {
  try {
    const resp = await apiFetch(`/api/scheduled-prints/${rec.id}`, {
      method: 'PUT',
      body: JSON.stringify(buildPayloadFromRecord(rec, { enabled: !rec.enabled }))
    }, () => emit('logout'))
    if (!resp.ok) throw new Error(await readError(resp))
    await loadRecords()
  } catch (e) {
    toast.add({ title: t('ui.m0c3b4cf7aa'), description: translateError(e.message), color: 'error', icon: 'i-lucide-x-circle' })
  }
}

async function confirmDelete(rec) {
  if (!confirm(t('ui.mb4c3a1301d', { p0: (rec.name || rec.filename) }))) return
  try {
    const resp = await apiFetch(`/api/scheduled-prints/${rec.id}`, { method: 'DELETE' }, () => emit('logout'))
    if (!resp.ok) throw new Error(await readError(resp))
    toast.add({ title: t('ui.m077a6d3771'), color: 'success', icon: 'i-lucide-check-circle' })
    await loadRecords()
  } catch (e) {
    toast.add({ title: t('ui.mc228558cf2'), description: translateError(e.message), color: 'error', icon: 'i-lucide-x-circle' })
  }
}

function buildPayloadFromRecord(rec, overrides = {}) {
  return {
    name: rec.name || '',
    printerUri: rec.printerUri,
    enabled: rec.enabled,
    isDuplex: rec.isDuplex,
    isColor: rec.isColor,
    copies: rec.copies || 1,
    orientation: rec.orientation || 'portrait',
    paperSize: rec.paperSize || 'A4',
    paperType: rec.paperType || 'plain',
    mediaSource: rec.mediaSource || 'auto',
    printScaling: rec.printScaling || 'fit',
    pageRange: rec.pageRange || '',
    pageSet: rec.pageSet || 'all',
    mirror: !!rec.mirror,
    watermarkText: rec.watermarkText || '',
    numberUp: rec.numberUp || 1,
    numberUpLayout: rec.numberUpLayout || 'lrtb',
    pageBorder: rec.pageBorder || 'none',
    scheduleType: rec.scheduleType,
    scheduleTime: rec.scheduleTime || '09:00',
    scheduleWeekday: rec.scheduleWeekday || 0,
    scheduleDay: rec.scheduleDay || 1,
    runAt: rec.runAt || '',
    ...overrides
  }
}

onMounted(async () => {
  await Promise.all([loadRecords(), loadPrinters()])
})
</script>
