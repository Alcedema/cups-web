<template>
  <div class="p-3 sm:p-4 md:p-6 space-y-4 md:space-y-6">
    <!-- 自动检测打印机 -->
    <UCard>
      <template #header>
        <h2 class="text-xl font-bold flex items-center gap-2">
          <UIcon name="i-lucide-scan-search" class="w-5 h-5" />
          {{ t('ui.md07e5519bb') }}
        </h2>
      </template>
      <div class="space-y-4">
        <UButton
          icon="i-lucide-scan-search"
          :loading="scanning"
          :disabled="scanning"
          @click="detectPrinters"
        >
          {{ t('ui.mb2af785744') }}
        </UButton>
        <div v-if="scanning" class="flex items-center gap-2 text-sm text-muted">
          <UIcon name="i-lucide-loader-circle" class="w-4 h-4 animate-spin" />
          {{ t('ui.m77823a7f1a') }}
        </div>
        <div v-if="detected.length" class="overflow-x-auto">
          <UTable :columns="detectColumns" :data="detected">
            <template #connection-cell="{ row }">
              <div class="flex items-center gap-1">
                <UIcon
                  :name="row.original.connection === 'usb' ? 'i-lucide-usb' : 'i-lucide-wifi'"
                  class="w-4 h-4"
                />
                <span>{{ row.original.connection === 'usb' ? 'USB' : t('ui.m97b31b5d63') }}</span>
              </div>
            </template>
            <template #printer-cell="{ row }">
              <div>{{ printerLabel(row.original) }}</div>
              <div class="text-xs text-muted truncate max-w-xs">{{ row.original.deviceUri }}</div>
            </template>
            <template #driverStatus-cell="{ row }">
              <div class="flex items-center gap-1 flex-wrap">
                <UBadge v-if="row.original.existingQueue" color="success" variant="subtle" size="sm">
                  {{ t('ui.m9aabde2b0d') }} {{ row.original.existingQueue }}
                </UBadge>
                <UBadge v-if="row.original.driverState === 'ready'" color="success" size="sm">
                  <UTooltip :text="row.original.topCandidate?.makeAndModel || ''">{{ t('ui.ma693aaad7e') }}</UTooltip>
                </UBadge>
                <UBadge v-else-if="row.original.driverState === 'driverless'" color="primary" size="sm">
                  {{ t('ui.mba82f23256') }}
                </UBadge>
                <UBadge v-else-if="row.original.driverState === 'needsVendorDriver'" color="warning" size="sm">
                  {{ t('ui.m39f5db4305') }} {{ row.original.driverMatch?.displayName || t('ui.m8d72392eda') }}
                </UBadge>
                <UBadge v-else-if="row.original.driverState === 'unmatched'" color="error" variant="subtle" size="sm">
                  {{ t('ui.m70e9828eed') }}
                </UBadge>
                <!-- 兼容旧后端（无 driverState 字段时退回三态） -->
                <template v-if="!row.original.driverState">
                  <UBadge v-if="row.original.hasDriver" color="success" size="sm">{{ t('ui.mab27f80d04') }}</UBadge>
                  <UBadge v-else-if="row.original.driverMatch" color="warning" size="sm">
                    {{ t('ui.m01ac58ef08') }} {{ row.original.driverMatch.displayName }}
                  </UBadge>
                  <UBadge v-else color="neutral" size="sm">{{ t('ui.m4d8c1c5b42') }}</UBadge>
                </template>
              </div>
            </template>
            <template #actions-cell="{ row }">
              <!-- 已添加队列：禁用操作 -->
              <UTooltip v-if="row.original.existingQueue" :text="t('ui.m4466ed906a')">
                <UButton size="sm" icon="i-lucide-check" color="neutral" variant="outline" disabled>
                  {{ t('ui.m889839915c') }}
                </UButton>
              </UTooltip>
              <!-- 推荐的驱动在当前架构上不可用 -->
              <UTooltip
                v-else-if="row.original.driverMatch && !row.original.hasDriver && !archSupported(row.original.driverMatch.arch)"
                :text="t('ui.m9223eb3e68', { p0: (currentArch) })"
              >
                <UButton size="sm" icon="i-lucide-ban" color="neutral" variant="outline" disabled>
                  {{ t('ui.m6d3c27126c') }}
                </UButton>
              </UTooltip>
              <!-- 需安装厂商驱动 -->
              <UButton
                v-else-if="row.original.driverMatch && !row.original.hasDriver"
                size="sm"
                icon="i-lucide-download"
                :loading="settingUp === row.original.deviceUri"
                :disabled="busy"
                @click="openPPDModal(row.original)"
              >
                {{ t('ui.m0183eda077') }}
              </UButton>
              <!-- 未匹配到驱动：手动选择 -->
              <UButton
                v-else-if="row.original.driverState === 'unmatched'"
                size="sm"
                variant="outline"
                icon="i-lucide-search"
                :loading="settingUp === row.original.deviceUri"
                :disabled="busy"
                @click="openPPDModal(row.original)"
              >
                {{ t('ui.mdea31c889f') }}
              </UButton>
              <!-- 已就绪 / driverless：直接添加 -->
              <UButton
                v-else
                size="sm"
                variant="outline"
                icon="i-lucide-plus"
                :loading="settingUp === row.original.deviceUri"
                :disabled="busy"
                @click="openPPDModal(row.original)"
              >
                {{ t('ui.mb32b4272f1') }}
              </UButton>
            </template>
          </UTable>
        </div>
        <div v-else-if="scanDone && !detected.length" class="text-sm text-muted">
          {{ t('ui.m74bdf88225') }}
        </div>
      </div>
    </UCard>

    <!-- 后台任务进度：安装/卸载/一键设置都是异步任务，这里展示实时日志 -->
    <UCard v-if="jobTitle">
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <h2 class="text-base font-semibold flex items-center gap-2">
            <UIcon
              :name="jobRunning ? 'i-lucide-loader-circle' : (jobFailed ? 'i-lucide-x-circle' : 'i-lucide-check-circle')"
              :class="['w-5 h-5', jobRunning && 'animate-spin', jobFailed && 'text-error']"
            />
            {{ jobTitle }}
          </h2>
          <UButton
            size="xs"
            variant="ghost"
            :icon="jobLogOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            @click="jobLogOpen = !jobLogOpen"
          >
            {{ jobLogOpen ? t('ui.mf40ebc6181') : t('ui.m5aa6d18542') }}
          </UButton>
        </div>
      </template>
      <div class="space-y-2">
        <p v-if="jobRunning" class="text-sm text-muted">
          {{ t('ui.mfc39b476ed') }}
        </p>
        <pre
          v-if="jobLogOpen"
          class="text-xs bg-elevated rounded p-3 max-h-64 overflow-auto whitespace-pre-wrap break-all"
        >{{ jobLog ? jobLog.split('\n').map(line => translateError(line)).join('\n') : t('ui.m976376ab86') }}</pre>
      </div>
    </UCard>

    <!-- 驱动管理 -->
    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2 flex-wrap">
          <h2 class="text-xl font-bold flex items-center gap-2">
            <UIcon name="i-lucide-puzzle" class="w-5 h-5" />
            {{ t('ui.mb26cfc8801') }}
          </h2>
          <UBadge v-if="currentArch" color="neutral" variant="subtle" size="sm">
            {{ t('ui.mf399247d94') }} {{ currentArch }}
          </UBadge>
        </div>
      </template>
      <div class="overflow-x-auto">
        <UTable :columns="driverColumns" :data="drivers">
          <template #description-cell="{ row }">
            <span>{{ translateError(row.original.description) }}</span>
            <UBadge v-if="row.original.needCompile" color="warning" size="xs" class="ml-1">{{ t('ui.m0faf74045b') }}</UBadge>
          </template>
          <template #arch-cell="{ row }">
            {{ (row.original.arch || []).join(', ') }}
          </template>
          <template #status-cell="{ row }">
            <div class="space-y-1">
              <UBadge v-if="row.original.installed" color="success" size="sm">
                {{ t('ui.ma8b6c39dca') }} {{ row.original.installedAt ? formatDate(row.original.installedAt) : '' }}
              </UBadge>
              <UBadge v-else color="neutral" size="sm">{{ t('ui.m156219e305') }}</UBadge>
              <!-- 驱动数据是挂载卷，换机器时可能与当前架构不符，必须提示重装 -->
              <UBadge
                v-if="row.original.installed && row.original.installedArch && currentArch && row.original.installedArch !== currentArch"
                color="warning"
                size="xs"
              >
                {{ t('ui.m0917afc151') }} {{ row.original.installedArch }}{{ t('ui.m108cbfcac5') }}
              </UBadge>
              <!--
                恢复方式徽章：厂商驱动普遍把产物装在 /opt、/usr/bin 等位置，只有归档了
                .deb 原件（package / hybrid）才能在容器重启后完整装回来。老快照没有
                restoreMode 字段，提示重装一次即可升级。
              -->
              <UTooltip
                v-if="row.original.installed && row.original.restoreMode"
                :text="restoreModeHint(row.original)"
              >
                <UBadge color="info" variant="subtle" size="xs">
                  {{ t('ui.m97e9f22d1c') }} {{ restoreModeLabel(row.original.restoreMode) }}
                  <template v-if="row.original.packageCount">
                    （{{ row.original.packageCount }} {{ t('ui.m010200ff29') }}
                  </template>
                </UBadge>
              </UTooltip>
              <UTooltip
                v-else-if="row.original.installed"
                :text="t('ui.m21e36e1e7b')"
              >
                <UBadge color="warning" variant="subtle" size="xs">
                  {{ t('ui.med1da30069') }}
                </UBadge>
              </UTooltip>
            </div>
          </template>
          <template #actions-cell="{ row }">
            <UTooltip
              v-if="!row.original.installed && row.original.supported === false"
              :text="t('ui.m788b573cc6', { p0: (currentArch) })"
            >
              <UButton size="sm" icon="i-lucide-ban" color="neutral" variant="outline" disabled>
                {{ t('ui.me8f88f51cc') }}
              </UButton>
            </UTooltip>
            <UTooltip
              v-else-if="!row.original.installed && row.original.hasScript === false"
              :text="t('ui.mb6a2dd2e41')"
            >
              <UButton size="sm" icon="i-lucide-ban" color="neutral" variant="outline" disabled>
                {{ t('ui.me8f88f51cc') }}
              </UButton>
            </UTooltip>
            <UButton
              v-else-if="!row.original.installed"
              size="sm"
              icon="i-lucide-download"
              :loading="installingDriver === row.original.name"
              :disabled="busy"
              @click="confirmInstall(row.original)"
            >
              {{ t('ui.me8f88f51cc') }}
            </UButton>
            <UButton
              v-else
              size="sm"
              variant="outline"
              color="error"
              icon="i-lucide-trash-2"
              :loading="removingDriver === row.original.name"
              :disabled="busy"
              @click="confirmRemove(row.original)"
            >
              {{ t('ui.m06bc14b60f') }}
            </UButton>
          </template>
        </UTable>
      </div>
    </UCard>

    <!-- 上传自定义驱动 -->
    <UCard>
      <template #header>
        <h2 class="text-xl font-bold flex items-center gap-2">
          <UIcon name="i-lucide-upload" class="w-5 h-5" />
          {{ t('ui.madafa18284') }}
        </h2>
      </template>
      <div class="space-y-3">
        <p class="text-sm text-muted">{{ t('ui.me570095a11') }}</p>
        <div class="flex flex-wrap items-center gap-3">
          <UButton variant="outline" icon="i-lucide-file-up" @click="triggerFileInput">
            {{ t('ui.m822fb37dba') }}
          </UButton>
          <span v-if="uploadFile" class="text-sm text-muted truncate max-w-xs">{{ uploadFile.name }}</span>
          <input
            ref="fileInputRef"
            type="file"
            accept=".ppd,.deb"
            class="hidden"
            @change="onFileSelected"
          />
        </div>
        <UButton
          v-if="uploadFile"
          color="primary"
          icon="i-lucide-upload"
          :loading="uploading"
          :disabled="uploading || busy"
          @click="uploadDriver"
        >
          {{ t('ui.m20a1af95f1') }}
        </UButton>

        <!-- .deb 无法自动恢复，必须显式告知，不能静默丢失 -->
        <UAlert
          v-if="customDebs.length"
          color="info"
          variant="subtle"
          icon="i-lucide-package"
          :title="t('ui.m370615efc1')"
        >
          <template #description>
            <p class="mb-1">{{ translateError(customDebNotice) }}</p>
            <ul class="list-disc pl-5">
              <li v-for="pkg in customDebs" :key="pkg.filename">
                {{ pkg.filename }}
                <span v-if="pkg.installedAt" class="text-muted">（{{ formatDate(pkg.installedAt) }}）</span>
              </li>
            </ul>
          </template>
        </UAlert>
      </div>
    </UCard>

    <!-- 安装确认弹窗 -->
    <UModal v-model:open="showInstallModal">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">{{ t('ui.ma0f42f891d') }}</h3>
          <p>{{ t('ui.m99a2c6821d') }} <strong>{{ pendingDriver?.displayName }}</strong>？</p>
          <p class="text-sm text-muted">{{ t('ui.mc877526dab') }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showInstallModal = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton color="primary" :loading="!!installingDriver" @click="installDriver">{{ t('ui.ma0f42f891d') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- 卸载确认弹窗 -->
    <UModal v-model:open="showRemoveModal">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">{{ t('ui.m4fec200ac3') }}</h3>
          <p>{{ t('ui.ma0e2b3bb74') }} <strong>{{ pendingDriver?.displayName }}</strong> {{ t('ui.m9d45d89439') }}</p>
          <p class="text-sm text-muted">{{ t('ui.m6b73957a14') }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showRemoveModal = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton color="error" :loading="!!removingDriver" @click="removeDriver">{{ t('ui.m4fec200ac3') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- PPD 候选选择弹窗 -->
    <UModal v-model:open="showPPDModal" :ui="{ width: 'max-w-lg' }">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">{{ t('ui.mba32358952') }}</h3>
          <!-- 设备摘要 -->
          <div class="text-sm">
            <div class="font-medium">{{ printerLabel(ppdModalPrinter) }}</div>
            <div class="text-xs text-muted truncate">{{ ppdModalPrinter?.deviceUri }}</div>
            <details v-if="ppdModalPrinter?.deviceId" class="mt-1">
              <summary class="text-xs text-muted cursor-pointer">{{ t('ui.m0a5f43d7c1') }}</summary>
              <code class="text-xs break-all">{{ ppdModalPrinter.deviceId }}</code>
            </details>
          </div>

          <!-- 已有队列警告 -->
          <UAlert
            v-if="ppdModalData?.existingQueue"
            color="warning"
            icon="i-lucide-alert-triangle"
            :title="t('ui.m34c8603474', { p0: (ppdModalData.existingQueue) })"
          />

          <!-- 候选查询失败降级 -->
          <UAlert
            v-if="ppdModalError"
            color="warning"
            icon="i-lucide-alert-triangle"
            :title="t('ui.m362110839d')"
          />

          <!-- 候选加载中 -->
          <div v-if="ppdModalLoading" class="space-y-2">
            <USkeleton v-for="i in 3" :key="i" class="h-12 w-full" />
          </div>

          <!-- 候选列表 -->
          <div v-else-if="ppdModalCandidates.length" class="space-y-1">
            <p class="text-xs text-muted">
              {{ t('ui.md8ae27cd60') }}
            </p>
            <URadioGroup v-model="selectedPPD" :items="ppdRadioItems">
              <template #label="{ item }">
                <div class="flex items-center gap-2 flex-wrap">
                  <span>{{ item.label }}</span>
                  <UBadge v-if="item.raw?.recommended" color="primary" size="xs">{{ t('ui.m1452deafc6') }}</UBadge>
                  <UBadge
                    :color="item.raw?.confidence === 'high' ? 'success' : item.raw?.confidence === 'medium' ? 'warning' : 'neutral'"
                    size="xs"
                  >
                    {{ item.raw?.confidence === 'high' ? t('ui.m6d42ca8a29') : item.raw?.confidence === 'medium' ? t('ui.ma2b8accae7') : t('ui.mbbb662f9c0') }}
                  </UBadge>
                  <UBadge v-if="item.raw?.driverdRank >= 1" color="primary" variant="subtle" size="xs">{{ t('ui.mafd0811b5b') }}</UBadge>
                </div>
                <div class="text-xs text-muted">{{ translateError(item.raw?.reason) }} · {{ item.raw?.makeAndModel }}</div>
              </template>
            </URadioGroup>

            <!-- IPP Everywhere 选项 -->
            <div class="border-t pt-2 mt-2">
              <UTooltip
                v-if="!ppdModalData?.driverless?.available"
                :text="ppdModalData?.driverless?.reason ? translateError(ppdModalData.driverless.reason) : t('ui.m7c53786065')"
              >
                <div class="opacity-50 cursor-not-allowed text-sm">
                  <input type="radio" disabled class="mr-2" />{{ t('ui.m526169f6b1') }}
                </div>
              </UTooltip>
              <label v-else class="flex items-center gap-2 text-sm cursor-pointer">
                <input
                  type="radio"
                  :checked="selectedPPD === 'everywhere'"
                  @change="selectedPPD = 'everywhere'"
                />
                {{ t('ui.m4a2905fc54') }}
              </label>
            </div>

            <!-- 高级选项：raw 队列 -->
            <details class="border-t pt-2 mt-2">
              <summary class="text-xs text-muted cursor-pointer">{{ t('ui.m91a61fdde8') }}</summary>
              <label class="flex items-center gap-2 text-sm cursor-pointer mt-1">
                <input
                  type="radio"
                  :checked="selectedPPD === '__raw__'"
                  @change="selectedPPD = '__raw__'"
                />
                {{ t('ui.m458f3ceac9') }}
              </label>
              <UAlert
                v-if="selectedPPD === '__raw__'"
                color="error"
                icon="i-lucide-alert-triangle"
                :title="t('ui.m92eb5baf5e')"
                :description="t('ui.m635e1a4d53')"
                class="mt-2"
              />
            </details>
          </div>

          <!-- 队列名 -->
          <div v-if="!ppdModalData?.existingQueue">
            <label class="text-sm font-medium">{{ t('ui.m45f34b3ed2') }}</label>
            <UInput v-model="ppdQueueName" class="mt-1" />
          </div>

          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showPPDModal = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton
              color="primary"
              :disabled="!!ppdModalData?.existingQueue || busy"
              :loading="settingUp === ppdModalPrinter?.deviceUri"
              @click="submitPPDSelection"
            >
              {{ ppdModalError ? t('ui.md4f0bea192') : t('ui.mec7ce6f381') }}
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { t, translateError } from '../i18n.js'

import { ref, computed, onMounted, onUnmounted } from 'vue'
import { apiFetch, readError } from '../utils/api'

defineProps({ session: Object })
const emit = defineEmits(['logout'])
const toast = useToast()

// 提交请求本身很快（后端立刻 202 返回 jobId），给个短超时即可
const SUBMIT_TIMEOUT = 30000
// 轮询间隔与总时长上限：后端任务硬超时是 30 分钟，前端留一点余量
const POLL_INTERVAL = 2000
const POLL_MAX_MS = 35 * 60 * 1000

// --- 后台任务（安装 / 卸载 / 一键设置统一走 /api/admin/drivers/jobs/{id} 轮询）---
const jobTitle = ref('')
const jobLog = ref('')
const jobLogOpen = ref(true)
const jobRunning = ref(false)
const jobFailed = ref(false)

let pollTimer = null
let unmounted = false

function startJobPanel(title) {
  jobTitle.value = title
  jobLog.value = ''
  jobRunning.value = true
  jobFailed.value = false
  jobLogOpen.value = true
}

function delay(ms) {
  return new Promise((resolve) => {
    pollTimer = setTimeout(resolve, ms)
  })
}

// 轮询任务直到 status !== 'running'，期间持续刷新日志；超时或组件卸载时抛错
async function pollDriverJob(jobId) {
  const deadline = Date.now() + POLL_MAX_MS
  while (Date.now() < deadline) {
    await delay(POLL_INTERVAL)
    if (unmounted) throw new Error(t('ui.mfb6253b032'))
    const resp = await apiFetch(`/api/admin/drivers/jobs/${jobId}`, {}, () => emit('logout'))
    if (!resp.ok) throw new Error(await readError(resp))
    const job = await resp.json()
    jobLog.value = job.log || ''
    if (job.status !== 'running') return job
  }
  throw new Error(t('ui.m4cdd793544'))
}

// 提交一个驱动任务并等待其结束，返回最终任务对象；提交失败/任务失败均抛错
async function runDriverJob(url, payload) {
  const resp = await apiFetch(url, {
    method: 'POST',
    body: JSON.stringify(payload),
    signal: AbortSignal.timeout(SUBMIT_TIMEOUT)
  }, () => emit('logout'))

  if (!resp.ok) {
    const msg = await readError(resp)
    // 409：已有驱动任务在跑（apt/dpkg 有全局锁，后端只允许一个任务）
    throw new Error(msg)
  }
  const data = await resp.json()
  if (!data.jobId) throw new Error(t('ui.m7384101318'))

  const job = await pollDriverJob(data.jobId)
  if (job.status !== 'succeeded') {
    throw new Error(translateError(job.error) || t('ui.ma3cfc7bf04'))
  }
  return job
}

function finishJobPanel(ok) {
  jobRunning.value = false
  jobFailed.value = !ok
}

// --- 自动检测打印机 ---
const scanning = ref(false)
const scanDone = ref(false)
const detected = ref([])
const settingUp = ref(null)

const detectColumns = [
  { id: 'connection', get header() { return t('ui.m485a26050c') } },
  { id: 'printer', get header() { return t('ui.m7d6376ef9f') } },
  { id: 'driverStatus', get header() { return t('ui.m9d1fc1aa2c') } },
  { id: 'actions', get header() { return t('ui.med31fbb483') } }
]

function printerLabel(printer) {
  const label = `${printer.manufacturer || ''} ${printer.model || ''}`.trim()
  return label || t('ui.mf6c6d345e1')
}

async function detectPrinters() {
  scanning.value = true
  scanDone.value = false
  detected.value = []
  candidatesByUri.value = {} // 重扫后旧候选一定失效
  try {
    const resp = await apiFetch('/api/admin/drivers/detect', {}, () => emit('logout'))
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.mb1ca0741b0'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    detected.value = (await resp.json()) || []
  } catch (e) {
    toast.add({ title: t('ui.mb1ca0741b0'), description: String(e), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    scanning.value = false
    scanDone.value = true
  }
}

// 请求体字段名与后端 adminSetupPrinterHandler 严格对齐。
async function setupPrinter(printer, opts = {}) {
  settingUp.value = printer.deviceUri
  startJobPanel(t('ui.mb85c9b728e', { p0: (printerLabel(printer)) }))
  try {
    const job = await runDriverJob('/api/admin/drivers/setup', {
      deviceUri: printer.deviceUri,
      driverName: printer.hasDriver ? '' : (printer.driverMatch?.name || ''),
      manufacturer: printer.manufacturer || '',
      model: printer.model || '',
      deviceId: printer.deviceId || '',
      ppdUri: opts.ppdUri || '',
      printerName: opts.printerName || '',
      allowRaw: opts.allowRaw || false
    })
    finishJobPanel(true)
    toast.add({
      title: t('ui.mc48405ac3a'),
      description: t('ui.m3a83d204e7', { p0: (job.result?.printerName || printerLabel(printer)) }),
      color: 'success',
      icon: 'i-lucide-check-circle'
    })
    await Promise.all([detectPrinters(), loadDrivers()])
  } catch (e) {
    finishJobPanel(false)
    toast.add({ title: t('ui.mbf76bc8195'), description: e.message || String(e), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    settingUp.value = null
  }
}

// --- PPD 候选选择 Modal ---
const showPPDModal = ref(false)
const ppdModalPrinter = ref(null)
const ppdModalData = ref(null)
const ppdModalLoading = ref(false)
const ppdModalError = ref('')
const ppdModalCandidates = ref([])
const selectedPPD = ref('')
const ppdQueueName = ref('')
// 独立状态 map，不挂 row.original（detected 是 ref 数组，给元素加属性易踩响应性坑）
const candidatesByUri = ref({})

const ppdRadioItems = computed(() =>
  ppdModalCandidates.value.map((c) => ({
    value: c.ppd,
    label: c.makeAndModel,
    raw: c
  }))
)

async function openPPDModal(printer) {
  ppdModalPrinter.value = printer
  ppdModalData.value = null
  ppdModalCandidates.value = []
  ppdModalError.value = ''
  selectedPPD.value = ''
  ppdQueueName.value = printer.suggestedName || ''
  showPPDModal.value = true

  // 同一 deviceUri 在本次扫描周期内只查一次
  if (candidatesByUri.value[printer.deviceUri]) {
    applyCandidateData(printer.deviceUri, candidatesByUri.value[printer.deviceUri])
    return
  }

  ppdModalLoading.value = true
  try {
    const params = new URLSearchParams({
      deviceUri: printer.deviceUri,
      deviceId: printer.deviceId || '',
      manufacturer: printer.manufacturer || '',
      model: printer.model || '',
      limit: '8'
    })
    const resp = await apiFetch(
      `/api/admin/drivers/ppds?${params}`,
      { signal: AbortSignal.timeout(20000) },
      () => emit('logout')
    )
    if (!resp.ok) {
      const msg = await readError(resp)
      if (resp.status === 429) {
        ppdModalError.value = t('ui.m9513c98c02')
      } else {
        ppdModalError.value = msg
      }
      return
    }
    const data = await resp.json()
    candidatesByUri.value[printer.deviceUri] = data
    applyCandidateData(printer.deviceUri, data)
  } catch (e) {
    ppdModalError.value = String(e)
  } finally {
    ppdModalLoading.value = false
  }
}

function applyCandidateData(uri, data) {
  ppdModalData.value = data
  ppdModalCandidates.value = data.candidates || []
  ppdQueueName.value = data.suggestedName || ppdQueueName.value
  // 默认选中 recommended 项，没有则选第一条
  const rec = ppdModalCandidates.value.find((c) => c.recommended)
  selectedPPD.value = rec ? rec.ppd : (ppdModalCandidates.value[0]?.ppd || '')
}

async function submitPPDSelection() {
  const printer = ppdModalPrinter.value
  if (!printer) return
  showPPDModal.value = false

  const opts = {
    ppdUri: ppdModalError.value ? '' : selectedPPD.value,
    printerName: ppdQueueName.value,
    allowRaw: selectedPPD.value === '__raw__'
  }
  await setupPrinter(printer, opts)
}

// --- 驱动管理 ---
const drivers = ref([])
const currentArch = ref('')
const customDebs = ref([])
const customDebNotice = ref('')
const installingDriver = ref(null)
const removingDriver = ref(null)
const pendingDriver = ref(null)
const showInstallModal = ref(false)
const showRemoveModal = ref(false)

// 任何一个驱动任务在跑时禁用所有触发按钮，防止重复点击（后端也会回 409）
const busy = computed(() => !!settingUp.value || !!installingDriver.value || !!removingDriver.value)

const driverColumns = [
  { accessorKey: 'displayName', get header() { return t('ui.m5455909dc9') } },
  { id: 'description', get header() { return t('ui.m4262c45dc7') } },
  { id: 'arch', get header() { return t('ui.m8b784b6288') } },
  { id: 'status', get header() { return t('ui.m6320b4a872') } },
  { id: 'actions', get header() { return t('ui.med31fbb483') } }
]

// 检测表格里的推荐驱动只有 arch 数组（DriverMeta），按当前架构自行判断是否可装
function archSupported(arch) {
  if (!arch || !arch.length) return true
  return arch.includes('all') || arch.includes(currentArch.value)
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

// restore_mode 由 scripts/driver/driver-install.sh 写进 metadata.txt：
//   package —— 全部产物都在归档的 .deb 里（最完整）
//   hybrid  —— 归档 .deb + 少量散落文件快照
//   files   —— 纯文件级快照（源码编译类驱动，产物都在白名单目录内）
function restoreModeLabel(mode) {
  switch (mode) {
    case 'package':
      return t('ui.m5f1780c040')
    case 'hybrid':
      return t('ui.m5cdf6d821f')
    case 'files':
      return t('ui.md3678902fe')
    default:
      return mode
  }
}

function restoreModeHint(row) {
  const n = row.packageCount || 0
  switch (row.restoreMode) {
    case 'package':
      return t('ui.m3287150c0c', { p0: (n) })
    case 'hybrid':
      return t('ui.mb766d0d703', { p0: (n) })
    case 'files':
      return t('ui.md7c73af682')
    default:
      return ''
  }
}

async function loadDrivers() {
  try {
    const resp = await apiFetch('/api/admin/drivers', {}, () => emit('logout'))
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.mc1bc5e880b'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    const data = await resp.json()
    drivers.value = (data.drivers || []).map(driver => ({ ...driver, get displayName() { return translateError(driver.displayName) }, get description() { return translateError(driver.description) } }))
    currentArch.value = data.currentArch || ''
    customDebs.value = data.customDebs || []
    customDebNotice.value = data.customDebNotice || ''
  } catch (e) {
    toast.add({ title: t('ui.mc1bc5e880b'), description: String(e), color: 'error', icon: 'i-lucide-x-circle' })
  }
}

function confirmInstall(driver) {
  pendingDriver.value = driver
  showInstallModal.value = true
}

function confirmRemove(driver) {
  pendingDriver.value = driver
  showRemoveModal.value = true
}

async function installDriver() {
  const driver = pendingDriver.value
  if (!driver) return
  installingDriver.value = driver.name
  showInstallModal.value = false
  startJobPanel(t('ui.md9fa616007', { p0: (driver.displayName) }))
  try {
    await runDriverJob('/api/admin/drivers/install', { name: driver.name })
    finishJobPanel(true)
    toast.add({ title: t('ui.mca637231de'), description: t('ui.m3febcf5706', { p0: (driver.displayName) }), color: 'success', icon: 'i-lucide-check-circle' })
    await loadDrivers()
  } catch (e) {
    finishJobPanel(false)
    toast.add({ title: t('ui.ma8581cc00f'), description: e.message || String(e), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    installingDriver.value = null
    pendingDriver.value = null
  }
}

async function removeDriver() {
  const driver = pendingDriver.value
  if (!driver) return
  removingDriver.value = driver.name
  showRemoveModal.value = false
  startJobPanel(t('ui.m298dafedcf', { p0: (driver.displayName) }))
  try {
    await runDriverJob('/api/admin/drivers/remove', { name: driver.name })
    finishJobPanel(true)
    toast.add({ title: t('ui.mb802a9dd9a'), description: t('ui.m8080dd4c19', { p0: (driver.displayName) }), color: 'success', icon: 'i-lucide-check-circle' })
    await loadDrivers()
  } catch (e) {
    finishJobPanel(false)
    toast.add({ title: t('ui.m2eb73d2d20'), description: e.message || String(e), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    removingDriver.value = null
    pendingDriver.value = null
  }
}

// --- 上传自定义驱动 ---
const fileInputRef = ref(null)
const uploadFile = ref(null)
const uploading = ref(false)

function triggerFileInput() {
  fileInputRef.value?.click()
}

function onFileSelected(e) {
  const file = e.target.files?.[0]
  if (file) {
    uploadFile.value = file
  }
}

async function uploadDriver() {
  if (!uploadFile.value) return
  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', uploadFile.value)
    const resp = await apiFetch('/api/admin/drivers/upload', {
      method: 'POST',
      body: formData,
      signal: AbortSignal.timeout(300000)
    }, () => emit('logout'))
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.m219481a6dd'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    const data = await resp.json()
    toast.add({
      title: t('ui.m844759c9f5'),
      // 正常情况下 .deb 会被归档并在容器重启时自动重装，没有 warning；
      // 只有归档失败（重启后恢复不了）时后端才给 warning，必须透传给用户
      description: data.warning
        ? t('ui.m3480f90b90', { p0: (uploadFile.value.name), p1: (translateError(data.warning)) })
        : t('ui.m297395aafd', { p0: (uploadFile.value.name) }),
      color: data.warning ? 'warning' : 'success',
      icon: data.warning ? 'i-lucide-triangle-alert' : 'i-lucide-check-circle'
    })
    if (data.log) {
      jobTitle.value = t('ui.mf7f2cd8f2c', { p0: (uploadFile.value.name) })
      jobLog.value = data.log
      jobRunning.value = false
      jobFailed.value = false
    }
    uploadFile.value = null
    if (fileInputRef.value) fileInputRef.value.value = ''
    await loadDrivers()
  } catch (e) {
    toast.add({ title: t('ui.m219481a6dd'), description: String(e), color: 'error', icon: 'i-lucide-x-circle' })
  } finally {
    uploading.value = false
  }
}

onMounted(() => {
  loadDrivers()
})

// 组件卸载时停掉轮询定时器，避免离开页面后仍在打接口
onUnmounted(() => {
  unmounted = true
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
})
</script>
