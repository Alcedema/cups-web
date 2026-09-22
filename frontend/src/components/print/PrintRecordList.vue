<template>
  <UCard>
    <template #header>
      <div class="flex items-center justify-between cursor-pointer select-none" @click="listExpanded = !listExpanded">
        <div class="flex items-center gap-2 font-semibold">
          <UIcon name="i-lucide-history" class="w-5 h-5" />
          {{ t('ui.mdff8b1cd27') }}
          <!-- 折叠时显示最近一条摘要 -->
          <span v-if="!listExpanded && records.length > 0" class="text-xs font-normal text-muted truncate max-w-48">
            — {{ records[0].filename }} · {{ formatTime(records[0].createdAt) }} · {{ statusText(records[0].status) }}
          </span>
        </div>
        <div class="flex items-center gap-1">
          <UButton :aria-label="t('accessibility.refresh')" variant="ghost" size="xs" icon="i-lucide-refresh-cw" @click.stop="$emit('refresh')" />
          <UIcon
            :name="listExpanded ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
            class="w-4 h-4 text-muted transition-transform duration-200"
          />
        </div>
      </div>
    </template>
    <div
      class="transition-all duration-300 ease-in-out overflow-hidden"
      :style="{ maxHeight: listExpanded ? '24rem' : '0px', visibility: listExpanded ? 'visible' : 'hidden' }"
    >
      <div class="space-y-2 max-h-96 overflow-y-auto">
        <div v-if="loading" class="text-center py-4">
          <UIcon name="i-lucide-loader-circle" class="w-5 h-5 animate-spin mx-auto text-muted" />
        </div>
        <div v-else-if="records.length === 0" class="text-center py-6 text-muted text-sm">
          {{ t('ui.mbcfa48fadc') }}
        </div>
        <div
          v-for="rec in records"
          :key="rec.id"
          class="border rounded-lg p-3 hover:shadow-sm transition cursor-pointer"
          @click="toggleRecord(rec.id)"
        >
          <div class="flex items-start gap-2">
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ rec.filename }}</p>
              <p class="text-xs text-muted mt-0.5">{{ recordPrinterName(rec.printerUri) }} · {{ rec.pages }}{{ t('ui.md24d3c9946') }}</p>
              <p class="text-xs text-muted">{{ formatTime(rec.createdAt) }}</p>
            </div>
            <UBadge :color="statusColor(rec.status)" variant="subtle" size="xs">
              {{ statusText(rec.status) }}
            </UBadge>
          </div>
          <!-- 展开详情 -->
          <div v-if="expandedRecords.has(rec.id)" class="mt-2 pt-2 border-t">
            <div class="grid grid-cols-2 gap-1 text-xs text-muted">
              <div><span class="font-medium">{{ t('ui.m87a3a198a4') }}</span>{{ rec.isColor ? t('ui.mdb57813f39') : t('ui.m47d88357c5') }}</div>
              <div><span class="font-medium">{{ t('ui.m6e861a2d46') }}</span>{{ rec.isDuplex ? t('ui.mb5141d3d19') : t('ui.m0c70665b6e') }}</div>
              <div><span class="font-medium">{{ t('ui.m495c96b458') }}</span>{{ rec.pages }}</div>
              <div v-if="rec.jobId"><span class="font-medium">{{ t('ui.m0ffc430a2a') }}</span>{{ rec.jobId }}</div>
            </div>
            <div class="mt-2 flex justify-end">
              <UButton
                size="xs"
                variant="outline"
                icon="i-lucide-printer"
                :loading="reprintingId === rec.id"
                @click.stop="openReprintDialog(rec)"
              >{{ t('ui.m7258742e12') }}</UButton>
            </div>
          </div>
        </div>
      </div>
    </div>

    <UModal v-model:open="showReprintModal" :ui="{ content: 'max-w-lg' }">
      <template #content>
        <div class="flex flex-col max-h-[85vh]">
          <div class="p-6 pb-3 border-b border-default shrink-0">
            <h3 class="text-lg font-semibold">{{ t('ui.m7258742e12') }}</h3>
            <div class="text-sm text-muted truncate mt-1">{{ t('ui.m9491ade278') }}{{ reprintRecord?.filename }}</div>
          </div>
          <div class="flex-1 overflow-y-auto p-6 space-y-4">
            <div>
              <label class="block text-sm font-medium mb-1">{{ t('ui.m7d6376ef9f') }}</label>
              <USelect
                v-model="reprintForm.printer"
                :items="printerSelectItems"
                value-key="value"
                label-key="label"
                :placeholder="t('ui.md97891fc1e')"
                class="w-full"
              />
            </div>
            <PrintOptions
              v-model:isColor="reprintForm.isColor"
              v-model:duplex="reprintForm.duplex"
              v-model:copies="reprintForm.copies"
              v-model:paperSize="reprintForm.paperSize"
              v-model:paperType="reprintForm.paperType"
              v-model:mediaSource="reprintForm.mediaSource"
              :media-source-supported="mediaSourceSupported"
              v-model:printScaling="reprintForm.printScaling"
              v-model:scalePercent="reprintForm.scalePercent"
              v-model:pageRange="reprintForm.pageRange"
              v-model:pageSet="reprintForm.pageSet"
              v-model:mirror="reprintForm.mirror"
              v-model:watermarkText="reprintForm.watermarkText"
              v-model:numberUp="reprintForm.numberUp"
              v-model:numberUpLayout="reprintForm.numberUpLayout"
              v-model:pageBorder="reprintForm.pageBorder"
            />
          </div>
          <div class="flex justify-end gap-2 p-6 pt-3 border-t border-default shrink-0">
            <UButton variant="ghost" @click="showReprintModal = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton color="primary" :loading="reprintingId != null" @click="submitReprint">{{ t('ui.m8cdeba8942') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </UCard>
</template>

<script setup>
import { t } from '../../i18n.js'

import { ref, computed } from 'vue'
import { formatTime, formatPrinterName, printerLabel, printerDescription, statusColor, statusText } from '../../utils/format'
import PrintOptions from './PrintOptions.vue'

const props = defineProps({
  records: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  printers: { type: Array, default: () => [] },
  currentPrinter: { type: String, default: '' },
  mediaSourceSupported: { type: Array, default: () => [] }
})

const emit = defineEmits(['refresh', 'reprint'])

const listExpanded = ref(window.innerWidth >= 1024)
const expandedRecords = ref(new Set())

const showReprintModal = ref(false)
const reprintingId = ref(null)
const reprintRecord = ref(null)

// 重打表单字段与 PrintOptions 组件保持完全一致（duplex 为字符串，isColor 为布尔）；
// 提交时再折算成后端 reprint 接口需要的 duplex/color 布尔值。
function defaultReprintForm() {
  return {
    printer: '',
    orientation: 'portrait',
    isColor: true,
    duplex: 'one-sided',
    copies: 1,
    paperSize: 'A4',
    paperType: 'plain',
    mediaSource: 'auto',
    printScaling: 'auto',
    scalePercent: 100,
    pageRange: '',
    pageSet: 'all',
    mirror: false,
    watermarkText: '',
    numberUp: 1,
    numberUpLayout: 'lrtb',
    pageBorder: 'none'
  }
}
const reprintForm = ref(defaultReprintForm())

const printerSelectItems = computed(() =>
  props.printers.map(p => ({
    label: printerLabel(p),
    description: printerDescription(p),
    value: p.uri
  }))
)

// recordPrinterName 用打印机列表把记录里的 URI 反查成「队列名 — 描述」。
// 记录落库的只有 URI，队列已删或列表还没加载时退回从 URI 取队列名。
const printersByUri = computed(() =>
  new Map(props.printers.map(p => [p.uri, p]))
)
function recordPrinterName(uri) {
  const p = printersByUri.value.get(uri)
  return p ? printerLabel(p) : formatPrinterName(uri)
}

function toggleRecord(id) {
  const s = new Set(expandedRecords.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  expandedRecords.value = s
}

function openReprintDialog(rec) {
  reprintRecord.value = rec
  const def = defaultReprintForm()
  // 用记录里持久化的完整参数精确预填第一次的设置；老记录缺字段则回落默认值（Issue #68）。
  reprintForm.value = {
    printer: props.currentPrinter || rec.printerUri,
    orientation: rec.orientation ?? def.orientation,
    isColor: rec.isColor,
    duplex: rec.isDuplex ? 'two-sided-long-edge' : 'one-sided',
    copies: rec.copies ?? def.copies,
    paperSize: rec.paperSize ?? def.paperSize,
    paperType: rec.paperType ?? def.paperType,
    mediaSource: rec.mediaSource ?? def.mediaSource,
    printScaling: /^\d+$/.test(rec.printScaling) ? 'custom' : (rec.printScaling ?? def.printScaling),
    scalePercent: /^\d+$/.test(rec.printScaling) ? Number(rec.printScaling) : def.scalePercent,
    pageRange: rec.pageRange ?? def.pageRange,
    pageSet: rec.pageSet ?? def.pageSet,
    mirror: rec.mirror ?? def.mirror,
    watermarkText: rec.watermarkText ?? def.watermarkText,
    numberUp: rec.numberUp ?? def.numberUp,
    numberUpLayout: rec.numberUpLayout ?? def.numberUpLayout,
    pageBorder: rec.pageBorder ?? def.pageBorder
  }
  showReprintModal.value = true
}

// 自定义缩放提交给后端的是纯数字字符串；输入框聚焦时可能残留区间外的中间值，这里兜底 clamp。
function clampScalePercent(v) {
  const n = Number(v)
  if (!Number.isFinite(n)) return '100'
  return String(Math.min(400, Math.max(10, Math.round(n))))
}

function submitReprint() {
  const rec = reprintRecord.value
  if (!rec) return
  const f = reprintForm.value
  reprintingId.value = rec.id
  showReprintModal.value = false
  emit('reprint', {
    id: rec.id,
    printer: f.printer,
    duplex: f.duplex !== 'one-sided',
    color: f.isColor,
    copies: f.copies,
    orientation: f.orientation,
    paperSize: f.paperSize,
    paperType: f.paperType,
    mediaSource: f.mediaSource,
    printScaling: f.printScaling === 'custom' ? clampScalePercent(f.scalePercent) : f.printScaling,
    pageRange: f.pageRange.trim(),
    pageSet: f.pageSet,
    mirror: f.mirror,
    watermarkText: f.watermarkText.trim(),
    numberUp: f.numberUp,
    numberUpLayout: f.numberUpLayout,
    pageBorder: f.pageBorder
  })
}

defineExpose({ clearReprintLoading: () => { reprintingId.value = null } })
</script>
