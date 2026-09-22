<template>
  <UCard>
    <template #header>
      <div class="flex items-center gap-2 font-semibold">
        <UIcon name="i-lucide-settings-2" class="w-5 h-5" />
        {{ t('ui.me63f16a4fb') }}
      </div>
    </template>
    <div class="space-y-4">
      <!-- ═══ 基础选项（始终显示） ═══ -->
      <!-- 颜色 -->
      <UFormField :label="t('ui.m2ee9066f2b')" :hint="isColor ? undefined : t('ui.m102efce124')">
        <div class="flex rounded-lg border border-muted overflow-hidden">
          <label v-for="item in colorItems" :key="String(item.value)"
            class="flex-1 flex items-center justify-center gap-1.5 py-2 px-2 cursor-pointer text-sm transition"
            :class="isColor === item.value ? 'bg-primary text-white font-medium' : 'hover:bg-elevated'">
            <input type="radio" :value="item.value" :checked="isColor === item.value" class="sr-only" @change="$emit('update:isColor', item.value)" />
            <UIcon :name="item.icon" class="w-3.5 h-3.5 shrink-0" />
            <span class="text-xs whitespace-nowrap">{{ item.label }}</span>
          </label>
        </div>
      </UFormField>

      <!-- 黑白反转（仅图片 + 黑白模式时显示，issue #87） -->
      <UFormField v-if="invertVisible" :label="t('ui.m138c10e9de')" :hint="t('ui.m12f500ea4b')">
        <label class="flex items-center gap-2 p-2 border rounded-lg cursor-pointer transition hover:bg-elevated w-fit"
          :class="invert ? 'border-primary bg-primary/5' : 'border-muted'">
          <UCheckbox :model-value="invert" @update:model-value="$emit('update:invert', $event)" />
          <UIcon name="i-lucide-contrast" class="w-4 h-4" />
          <span class="text-sm">{{ t('ui.m1011ab280f') }}</span>
        </label>
      </UFormField>

      <!-- 双面 + 份数 -->
      <div class="grid grid-cols-2 gap-3">
        <UFormField :label="t('ui.m8b1a58c572')">
          <USelect :model-value="duplex" :items="duplexItems" value-key="value" label-key="label" class="w-full" @update:model-value="$emit('update:duplex', $event)" />
        </UFormField>

        <UFormField :label="t('ui.mdb9aa2283c')">
          <UInput
            :model-value="copies"
            type="number"
            :min="1"
            :max="99"
            class="w-full"
            @update:model-value="$emit('update:copies', Number($event))"
          />
        </UFormField>
      </div>

      <!-- ═══ 高级选项折叠区 ═══ -->
      <div class="border-t border-default pt-2">
        <button
          type="button"
          class="flex items-center gap-1.5 w-full text-xs sm:text-sm text-primary hover:text-primary/80 transition cursor-pointer py-1"
          @click="showAdvanced = !showAdvanced"
        >
          <UIcon
            name="i-lucide-chevron-right"
            class="w-3.5 h-3.5 transition-transform duration-200 shrink-0"
            :class="showAdvanced ? 'rotate-90' : ''"
          />
          <span class="font-medium">{{ t('ui.m91a61fdde8') }}</span>
          <span v-if="!showAdvanced" class="text-[11px] sm:text-xs text-muted ml-1 truncate">{{ advancedSummary }}</span>
        </button>

        <div
          class="overflow-hidden transition-all duration-300 ease-in-out"
          :style="{ maxHeight: showAdvanced ? '1000px' : '0px', opacity: showAdvanced ? 1 : 0, visibility: showAdvanced ? 'visible' : 'hidden' }"
        >
          <div class="space-y-4 pt-3">
            <!-- 纸张大小 + 纸张类型 -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <UFormField :label="t('ui.m37e1086195')">
                <USelect :model-value="paperSize" :items="paperSizeItems" value-key="value" label-key="label" class="w-full" @update:model-value="$emit('update:paperSize', $event)" />
              </UFormField>
              <UFormField :label="t('ui.m67340afa03')">
                <USelect :model-value="paperType" :items="paperTypeItems" value-key="value" label-key="label" class="w-full" @update:model-value="$emit('update:paperType', $event)" />
              </UFormField>
            </div>

            <!-- 进纸盒（仅当打印机上报可用纸盒时显示） -->
            <UFormField v-if="mediaSourceItems.length > 1" :label="t('ui.m321aeb11e3')" :hint="t('ui.m07ad91e27e')">
              <USelect :model-value="mediaSource" :items="mediaSourceItems" value-key="value" label-key="label" class="w-full" @update:model-value="$emit('update:mediaSource', $event)" />
            </UFormField>

            <!-- 缩放 + 页面范围 -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <UFormField :label="t('ui.mb762aec769')">
                <div class="flex gap-2">
                  <USelect :model-value="printScaling" :items="scalingItems" value-key="value" label-key="label" :class="printScaling === 'custom' ? 'w-28' : 'w-full'" @update:model-value="$emit('update:printScaling', $event)" />
                  <div v-if="printScaling === 'custom'" class="flex items-center gap-1 flex-1">
                    <UInput
                      type="number"
                      :model-value="scalePercent"
                      :min="10"
                      :max="400"
                      :step="5"
                      class="w-full"
                      @update:model-value="onScalePercentInput"
                      @blur="onScalePercentBlur"
                    />
                    <span class="text-sm text-gray-500 shrink-0">%</span>
                  </div>
                </div>
              </UFormField>
              <UFormField :label="t('ui.macfeb82ed6')" :hint="pageRangeError || t('ui.mdd8df0fd8d')">
                <UInput
                  :model-value="pageRange"
                  :placeholder="t('ui.mfea4d27ea3')"
                  class="w-full"
                  :color="pageRangeError ? 'error' : undefined"
                  @update:model-value="onPageRangeInput"
                />
              </UFormField>
            </div>

            <!-- 一张多页（N-up） -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <UFormField :label="t('ui.md39e754f3c')" :hint="t('ui.m502888d4c2')">
                <USelect :model-value="numberUp" :items="numberUpItems" value-key="value" label-key="label" class="w-full" @update:model-value="$emit('update:numberUp', Number($event))" />
              </UFormField>
              <UFormField v-if="numberUp > 1" :label="t('ui.m86329b52b3')">
                <USelect :model-value="numberUpLayout" :items="numberUpLayoutItems" value-key="value" label-key="label" class="w-full" @update:model-value="$emit('update:numberUpLayout', $event)" />
              </UFormField>
            </div>
            <UFormField v-if="numberUp > 1" :label="t('ui.ma091658f08')">
              <label class="flex items-center gap-2 p-2 border rounded-lg cursor-pointer transition hover:bg-elevated w-fit"
                :class="pageBorder === 'single' ? 'border-primary bg-primary/5' : 'border-muted'">
                <UCheckbox :model-value="pageBorder === 'single'" @update:model-value="$emit('update:pageBorder', $event ? 'single' : 'none')" />
                <UIcon name="i-lucide-square" class="w-4 h-4" />
                <span class="text-sm">{{ t('ui.m79c4673b1a') }}</span>
              </label>
            </UFormField>

            <!-- 页面子集（手动双面 / 分册排版） -->
            <UFormField :label="t('ui.md9e73e9750')" :hint="pageSetHint">
              <div class="flex rounded-lg border border-muted overflow-hidden">
                <label
                  v-for="item in pageSetItems"
                  :key="item.value"
                  class="flex-1 flex items-center justify-center gap-1.5 py-2 px-2 cursor-pointer text-sm transition"
                  :class="pageSet === item.value ? 'bg-primary text-white font-medium' : 'hover:bg-elevated'"
                >
                  <input type="radio" :value="item.value" :checked="pageSet === item.value" class="sr-only" @change="onPageSetChange(item.value)" />
                  <UIcon :name="item.icon" class="w-3.5 h-3.5 shrink-0" />
                  <span class="text-xs whitespace-nowrap">{{ item.label }}</span>
                </label>
              </div>
            </UFormField>

            <!-- 镜像打印 -->
            <UFormField :label="t('ui.m311f63c825')">
              <label class="flex items-center gap-2 p-2 border rounded-lg cursor-pointer transition hover:bg-elevated w-fit"
                :class="mirror ? 'border-primary bg-primary/5' : 'border-muted'">
                <UCheckbox :model-value="mirror" @update:model-value="$emit('update:mirror', $event)" />
                <UIcon name="i-lucide-flip-horizontal" class="w-4 h-4" />
                <span class="text-sm">{{ t('ui.mc582938da1') }}</span>
              </label>
            </UFormField>

            <!-- 水印文字 -->
            <UFormField :label="t('ui.m454aca5e71')" :hint="t('ui.m39542bbade')">
              <UInput
                :model-value="watermarkText"
                :placeholder="t('ui.m104bb76bb5')"
                class="w-full"
                @update:model-value="$emit('update:watermarkText', $event)"
              />
            </UFormField>
          </div>
        </div>
      </div>

    </div>
  </UCard>
</template>

<script setup>
import { t } from '../../i18n.js'

import { ref, computed, watch } from 'vue'

const props = defineProps({
  isColor: { type: Boolean, default: true },
  invert: { type: Boolean, default: false },
  invertVisible: { type: Boolean, default: false },
  duplex: { type: String, default: 'one-sided' },
  copies: { type: Number, default: 1 },
  paperSize: { type: String, default: 'A4' },
  paperType: { type: String, default: 'plain' },
  mediaSource: { type: String, default: 'auto' },
  mediaSourceSupported: { type: Array, default: () => [] },
  printScaling: { type: String, default: 'auto' },
  scalePercent: { type: Number, default: 100 },
  pageRange: { type: String, default: '' },
  pageSet: { type: String, default: 'all' },
  mirror: { type: Boolean, default: false },
  watermarkText: { type: String, default: '' },
  numberUp: { type: Number, default: 1 },
  numberUpLayout: { type: String, default: 'lrtb' },
  pageBorder: { type: String, default: 'none' },
  printing: { type: Boolean, default: false }
})

const emit = defineEmits([
  'update:isColor', 'update:invert',
  'update:duplex', 'update:copies',
  'update:paperSize', 'update:paperType', 'update:mediaSource', 'update:printScaling', 'update:scalePercent', 'update:pageRange',
  'update:pageSet', 'update:mirror', 'update:watermarkText',
  'update:numberUp', 'update:numberUpLayout', 'update:pageBorder'
])

const showAdvanced = ref(localStorage.getItem('print_options_expanded') === '1')
watch(showAdvanced, (val) => { localStorage.setItem('print_options_expanded', val ? '1' : '0') })
const pageRangeError = ref('')

// IPP media-source keyword → 中文名映射。不同打印机上报的纸盒关键字差异很大，
// 未命中的关键字（如 tray-3）会走 mediaSourceLabel 的通用规则或原样显示。
const mediaSourceNames = {
  'auto': t('ui.mdb9773faa3'),
  'auto-select': t('ui.mdb9773faa3'),
  'main': t('ui.m915b644f8b'),
  'alternate': t('ui.me78838da7a'),
  'large-capacity': t('ui.m61362e1a24'),
  'manual': t('ui.mdc8a4fa242'),
  'bypass': t('ui.m0bd1c3caac'),
  'by-pass-tray': t('ui.m0bd1c3caac'),
  'multipurpose': t('ui.m461b4d0b34'),
  'envelope': t('ui.md19eedbeca'),
  'top': t('ui.m8555320a3b'),
  'middle': t('ui.m785ab9628a'),
  'bottom': t('ui.mff0acb09d4'),
  'left': t('ui.m908a9aef93'),
  'right': t('ui.m5540dfd954'),
  'center': t('ui.m6c7e554381'),
  'rear': t('ui.m521f282dbb'),
  'side': t('ui.ma68f4e4de6'),
  'photo': t('ui.m13dc485ee8'),
  'hagaki': t('ui.m18dd2d47a2'),
  'disc': t('ui.m840f533ec5')
}

function mediaSourceLabel(key) {
  if (mediaSourceNames[key]) return mediaSourceNames[key]
  // tray-1 / tray-2 ... → 纸盒 1 / 纸盒 2
  const m = /^tray-?(\d+)$/i.exec(key)
  if (m) return t('ui.m9603ec58ef', { p0: (m[1]) })
  return key
}

// 供 USelect 使用的纸盒选项：始终含「自动选择」，其余来自打印机上报的 media-source-supported。
const mediaSourceItems = computed(() => {
  const items = [{ get label() { return t('ui.mdb9773faa3') }, value: 'auto' }]
  for (const key of props.mediaSourceSupported) {
    if (key === 'auto' || key === 'auto-select') continue
    items.push({ label: mediaSourceLabel(key), value: key })
  }
  return items
})

const advancedSummary = computed(() => {
  const sizeLabel = paperSizeItems.find(i => i.value === props.paperSize)?.label?.split(' ')[0] || props.paperSize
  const typeLabel = paperTypeItems.find(i => i.value === props.paperType)?.label || props.paperType
  const scaleLabel = scalingItems.find(i => i.value === props.printScaling)?.label || props.printScaling
  const parts = [sizeLabel, typeLabel, scaleLabel]
  if (props.mediaSource && props.mediaSource !== 'auto') parts.push(mediaSourceLabel(props.mediaSource))
  if (props.pageRange) parts.push(t('ui.md834c80a43', { p0: (props.pageRange) }))
  const pageSetLabel = pageSetItems.find(i => i.value === props.pageSet)?.label
  if (props.pageSet && props.pageSet !== 'all' && pageSetLabel) parts.push(pageSetLabel)
  if (props.numberUp > 1) {
    parts.push(t('ui.mff8eeef108', { p0: (props.numberUp) }))
    if (props.pageBorder === 'single') parts.push(t('ui.mf1d17dc26f'))
  }
  if (props.mirror) parts.push(t('ui.m176c09844e'))
  if (props.watermarkText) parts.push(t('ui.m7185ebd0fd', { p0: (props.watermarkText) }))
  return parts.join(' / ')
})

const colorItems = [
  { get label() { return t('ui.mb812ee08df') }, value: true, icon: 'i-lucide-palette' },
  { get label() { return t('ui.m83c437ef48') }, value: false, icon: 'i-lucide-contrast' }
]

const duplexItems = [
  { get label() { return t('ui.mb642ee0ed4') }, value: 'one-sided' },
  { get label() { return t('ui.m6f7cb1308f') }, value: 'two-sided-long-edge' },
  { get label() { return t('ui.mbfd81c2993') }, value: 'two-sided-short-edge' }
]

const paperSizeItems = [
  { label: 'A5 (148×210mm)', value: 'A5' },
  { label: 'A4 (210×297mm)', value: 'A4' },
  { label: 'A3 (297×420mm)', value: 'A3' },
  { label: 'A2 (420×594mm)', value: 'A2' },
  { label: 'A1 (594×841mm)', value: 'A1' },
  { get label() { return t('ui.mb8abfc1620') }, value: '5inch' },
  { get label() { return t('ui.me6af69a8b8') }, value: '6inch' },
  { get label() { return t('ui.macead90376') }, value: '7inch' },
  { get label() { return t('ui.m2fdd6b5a8a') }, value: '8inch' },
  { get label() { return t('ui.mf0f85103e8') }, value: '10inch' },
  { label: 'Letter (8.5×11in)', value: 'Letter' },
  { label: 'Legal (8.5×14in)', value: 'Legal' }
]

const paperTypeItems = [
  { get label() { return t('ui.m66dbc1464f') }, value: 'plain' },
  { get label() { return t('ui.md8aeeee56f') }, value: 'photo' },
  { get label() { return t('ui.m13e79a5a4d') }, value: 'glossy' },
  { get label() { return t('ui.mc81a6873a8') }, value: 'matte' },
  { get label() { return t('ui.m5835fc3d91') }, value: 'envelope' },
  { get label() { return t('ui.m1dbc7ef297') }, value: 'cardstock' },
  { get label() { return t('ui.m226b062885') }, value: 'labels' },
  { get label() { return t('ui.mdb9773faa3') }, value: 'auto' }
]

const scalingItems = [
  { get label() { return t('ui.m7eb336e42c') }, value: 'auto' },
  { get label() { return t('ui.mb58732dbe0') }, value: 'auto-fit' },
  { get label() { return t('ui.ma174176bb0') }, value: 'fit' },
  { get label() { return t('ui.mffe02a50f3') }, value: 'fill' },
  { get label() { return t('ui.mf71faab4dc') }, value: 'none' },
  { get label() { return t('ui.m4eafa9e925') }, value: 'custom' }
]

const pageSetItems = [
  { get label() { return t('ui.mc5bed289c5') }, value: 'all', icon: 'i-lucide-copy' },
  { get label() { return t('ui.md1084a2a27') }, value: 'odd', icon: 'i-lucide-list-ordered' },
  { get label() { return t('ui.m2db9e2637d') }, value: 'even', icon: 'i-lucide-list-ordered' },
  { get label() { return t('ui.mb1b1045d21') }, value: 'even-reverse', icon: 'i-lucide-arrow-down-up' }
]

const numberUpItems = [
  { get label() { return t('ui.meda7587d47') }, value: 1 },
  { get label() { return t('ui.m235cc2a006') }, value: 2 },
  { get label() { return t('ui.mf2e346eba4') }, value: 4 },
  { get label() { return t('ui.m6cf2aa240e') }, value: 6 },
  { get label() { return t('ui.m34c8187819') }, value: 9 },
  { get label() { return t('ui.meaaadb767f') }, value: 16 }
]

const numberUpLayoutItems = [
  { get label() { return t('ui.m9c3fddbc00') }, value: 'lrtb' },
  { get label() { return t('ui.m51a02ec417') }, value: 'rltb' },
  { get label() { return t('ui.mf8a8615424') }, value: 'tblr' },
  { get label() { return t('ui.mcb44b4c17a') }, value: 'tbrl' }
]

// 输入过程中只夹上限：若这里连下限一起夹，用户想输 40 时刚敲下 "4" 就会被弹成 10，
// 后面再敲 "0" 就变成 100。下限留到 blur 时归一。
function onScalePercentInput(val) {
  const n = Number(val)
  if (!Number.isFinite(n)) return
  emit('update:scalePercent', Math.min(400, Math.max(0, Math.round(n))))
}

function onScalePercentBlur() {
  const n = Number(props.scalePercent)
  const fixed = Number.isFinite(n) ? Math.min(400, Math.max(10, Math.round(n))) : 100
  if (fixed !== props.scalePercent) emit('update:scalePercent', fixed)
}

// 页面子集切换：odd / even 是"手动双面"用法，语义上必须单面输出；如果此时
// 双面仍是 two-sided-*，CUPS pdftopdf 会用空白页填补被过滤掉的另一面，产生
// 间隔空白页（issue #109）。选中奇/偶时联动把 duplex 拉回 one-sided，同时
// 用 hint 告知用户。even-reverse 也一样：它由后端把偶数页 PDF 层重排后再送
// CUPS，链路语义仍是单面输出，不能带双面。
function onPageSetChange(val) {
  emit('update:pageSet', val)
  const manualDuplex = val === 'odd' || val === 'even' || val === 'even-reverse'
  if (manualDuplex && props.duplex !== 'one-sided') {
    emit('update:duplex', 'one-sided')
  }
}

const pageSetHint = computed(() => {
  if (props.pageSet === 'odd' || props.pageSet === 'even' || props.pageSet === 'even-reverse') {
    return t('ui.m78ae98c3c9')
  }
  return t('ui.m8b79bb0b92')
})

function onPageRangeInput(val) {
  emit('update:pageRange', val)
  validatePageRange(val)
}

function validatePageRange(val) {
  if (typeof val !== 'string') val = ''
  val = val.trim()
  if (!val) { pageRangeError.value = ''; return }

  const normalizedVal = val
    .replace(/[－—–―]/g, '-')
    .replace(/\s*-\s*/g, '-')
    .replace(/[，,]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()

  if (normalizedVal !== val) {
    emit('update:pageRange', normalizedVal)
    val = normalizedVal
  }

  const pattern = /^(\d+(-\d+)?)(\s+\d+(-\d+)?)*$/
  pageRangeError.value = pattern.test(val) ? '' : t('ui.m10c8f192a9')
}
</script>
