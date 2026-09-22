<template>
  <div class="p-3 sm:p-4 md:p-6 max-w-5xl mx-auto space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <h1 class="text-lg font-bold flex items-center gap-2">
        <UIcon name="i-lucide-key-round" class="w-5 h-5 text-primary" />
        {{ t('ui.m5f600b307b') }}
      </h1>
      <div class="flex items-center gap-2">
        <UButton
          variant="ghost"
          size="xs"
          icon="i-lucide-refresh-cw"
          :loading="loading"
          @click="loadKeys"
        >{{ t('ui.maee8874341') }}</UButton>
        <UButton
          size="sm"
          color="primary"
          icon="i-lucide-plus"
          @click="openCreate"
        >{{ t('ui.md9433725d6') }}</UButton>
      </div>
    </div>

    <UAlert
      color="neutral"
      variant="soft"
      icon="i-lucide-info"
      :title="t('ui.mcb4ba40bf8')"
      :description="t('ui.m1c1f3c40af')"
    />

    <div v-if="loading" class="flex items-center justify-center py-10 text-muted gap-2">
      <UIcon name="i-lucide-loader-circle" class="w-5 h-5 animate-spin" />
      {{ t('ui.m4927a53bcc') }}
    </div>
    <div v-else-if="!keys.length" class="text-center py-16 text-muted">
      <UIcon name="i-lucide-key-off" class="w-10 h-10 mx-auto mb-2 opacity-40" />
      <div>{{ t('ui.m6c008626d4') }}</div>
    </div>
    <div v-else class="space-y-3">
      <UCard v-for="k in keys" :key="k.id" class="overflow-hidden">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div class="min-w-0 space-y-1">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="font-medium truncate">{{ k.name }}</span>
              <UBadge color="neutral" variant="outline" size="sm">
                <code class="font-mono">{{ k.prefix }}</code>
              </UBadge>
              <UBadge
                v-if="k.expiresAt"
                :color="isExpired(k) ? 'error' : 'warning'"
                variant="soft"
                size="sm"
              >
                {{ isExpired(k) ? t('ui.m2fe0e3339a') : t('ui.m49f72586d7', { p0: (formatDate(k.expiresAt)) }) }}
              </UBadge>
              <UBadge v-else color="success" variant="soft" size="sm">{{ t('ui.m2c60316d5e') }}</UBadge>
            </div>
            <div class="text-xs text-muted space-y-0.5">
              <div>{{ t('ui.mdc4d662d07') }} {{ formatDate(k.createdAt) }}</div>
              <div v-if="k.lastUsedAt">
                {{ t('ui.m39a9046f9c') }} {{ formatDate(k.lastUsedAt) }}
                <span v-if="k.lastUsedIp" class="ml-1 text-default/50">{{ t('extra.from_address', { address: k.lastUsedIp }) }}</span>
              </div>
              <div v-else class="text-default/50">{{ t('ui.m9bd11dda28') }}</div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <UButton
              size="xs"
              variant="outline"
              color="error"
              icon="i-lucide-trash-2"
              @click="confirmDelete(k)"
            >{{ t('ui.m2f9daa8289') }}</UButton>
          </div>
        </div>
      </UCard>
    </div>

    <!-- 新建密钥表单 -->
    <UModal v-model:open="showCreate">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold flex items-center gap-2">
            <UIcon name="i-lucide-key-round" class="w-5 h-5 text-primary" />
            {{ t('ui.m6a5c7ab7cf') }}
          </h3>
          <div class="space-y-3">
            <div>
              <label class="block text-sm font-medium mb-1">{{ t('ui.m36647d5ec6') }}</label>
              <UInput v-model="createForm.name" :placeholder="t('ui.m7a862a325a')" maxlength="64" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">{{ t('ui.m9c2a28e8f9') }}</label>
              <USelect
                v-model="createForm.expiresInDays"
                :items="expiryOptions"
                value-key="value"
                label-key="label"
              />
              <p class="text-xs text-muted mt-1">{{ t('ui.m8c9bd6488c') }}</p>
            </div>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" @click="showCreate = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton
              color="primary"
              :loading="creating"
              :disabled="!createForm.name.trim() || creating"
              @click="submitCreate"
            >{{ t('ui.mcde2cd071d') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <!-- 明文密钥展示（仅一次） -->
    <UModal v-model:open="showReveal" :dismissible="false">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold flex items-center gap-2 text-warning">
            <UIcon name="i-lucide-alert-triangle" class="w-5 h-5" />
            {{ t('ui.m1429438e6c') }}
          </h3>
          <p class="text-sm text-muted">
            {{ t('ui.m256cc401ee') }}
          </p>
          <div class="p-3 rounded bg-elevated/60 border border-default">
            <div class="flex items-center gap-2">
              <code class="font-mono text-sm break-all flex-1 select-all">{{ revealedKey }}</code>
              <UButton
                size="xs"
                variant="outline"
                icon="i-lucide-copy"
                @click="copyKey"
              >{{ t('ui.m63d90d9773') }}</UButton>
            </div>
          </div>
          <div class="text-xs text-muted space-y-1">
            <div>{{ t('ui.md997539070') }}</div>
            <pre class="p-2 rounded bg-elevated/60 border border-default overflow-x-auto"><code>curl -H "Authorization: Bearer {{ revealedKey }}" \
     {{ origin }}/api/printers</code></pre>
          </div>
          <div class="flex justify-end">
            <UButton color="primary" @click="closeReveal">{{ t('ui.m0e1df87d77') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="showDelete">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">{{ t('ui.ma3ea3c17b4') }}</h3>
          <p>{{ t('ui.md1b77c7c5a') }} <strong>{{ pendingDelete?.name }}</strong>（{{ pendingDelete?.prefix }}{{ t('ui.md8671711e8') }}</p>
          <p class="text-sm text-muted">{{ t('ui.meeef9685a2') }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showDelete = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton color="error" :loading="deleting" @click="executeDelete">{{ t('ui.ma3ea3c17b4') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { t, formatLocale, formatNumber } from '../i18n.js'

import { ref, computed, onMounted } from 'vue'
import { apiFetch, readError } from '../utils/api'

const emit = defineEmits(['logout'])
const toast = useToast()

const loading = ref(false)
const keys = ref([])
const showCreate = ref(false)
const creating = ref(false)
const createForm = ref({ name: '', expiresInDays: 0 })
const showReveal = ref(false)
const revealedKey = ref('')
const showDelete = ref(false)
const deleting = ref(false)
const pendingDelete = ref(null)

const origin = computed(() => (typeof window !== 'undefined' ? window.location.origin : ''))

const expiryOptions = [
  { get label() { return t('ui.m2c60316d5e') }, value: 0 },
  { get label() { return t('ui.m38eefacbb3') }, value: 7 },
  { get label() { return t('ui.m84ad2952a3') }, value: 30 },
  { get label() { return t('ui.mcb82f41919') }, value: 90 },
  { get label() { return t('ui.m3226061bb8') }, value: 365 }
]

function formatDate(v) {
  if (!v) return ''
  try {
    const d = new Date(v)
    if (Number.isNaN(d.getTime())) return v
    return d.toLocaleString(formatLocale())
  } catch {
    return v
  }
}

function isExpired(k) {
  if (!k.expiresAt) return false
  const d = new Date(k.expiresAt)
  if (Number.isNaN(d.getTime())) return false
  return d.getTime() < Date.now()
}

async function loadKeys() {
  loading.value = true
  try {
    const resp = await apiFetch('/api/api-keys', {}, () => emit('logout'))
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.md1d044826a'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    keys.value = await resp.json()
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.value = { name: '', expiresInDays: 0 }
  showCreate.value = true
}

async function submitCreate() {
  const name = createForm.value.name.trim()
  if (!name) return
  creating.value = true
  try {
    const resp = await apiFetch('/api/api-keys', {
      method: 'POST',
      body: JSON.stringify({
        name,
        expiresInDays: Number(createForm.value.expiresInDays) || 0
      })
    }, () => emit('logout'))
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.m7e6a71efbf'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    const data = await resp.json()
    revealedKey.value = data.key || ''
    showCreate.value = false
    showReveal.value = true
    await loadKeys()
  } finally {
    creating.value = false
  }
}

async function copyKey() {
  try {
    await navigator.clipboard.writeText(revealedKey.value)
    toast.add({ title: t('ui.m8f6f8d979c'), color: 'success', icon: 'i-lucide-check-circle' })
  } catch {
    toast.add({ title: t('ui.m753d8bb0da'), get description() { return t('ui.m06d95da71f') }, color: 'warning', icon: 'i-lucide-alert-triangle' })
  }
}

function closeReveal() {
  revealedKey.value = ''
  showReveal.value = false
}

function confirmDelete(k) {
  pendingDelete.value = k
  showDelete.value = true
}

async function executeDelete() {
  const k = pendingDelete.value
  if (!k) return
  deleting.value = true
  try {
    const resp = await apiFetch(`/api/api-keys/${k.id}`, { method: 'DELETE' }, () => emit('logout'))
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.mc228558cf2'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    toast.add({ title: t('ui.m5223f91b96'), color: 'success', icon: 'i-lucide-check-circle' })
    await loadKeys()
  } finally {
    deleting.value = false
    showDelete.value = false
    pendingDelete.value = null
  }
}

onMounted(loadKeys)
</script>
