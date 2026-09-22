<template>
  <div class="p-3 sm:p-4 md:p-6 space-y-4 md:space-y-6">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 md:gap-6">
      <UCard>
        <template #header>
          <h2 class="text-xl font-bold flex items-center gap-2">
            <UIcon name="i-lucide-users" class="w-5 h-5" />
            {{ t('ui.mfbf413d429') }}
          </h2>
        </template>
        <UForm @submit="saveUser" :state="form" class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <UInput v-model="form.username" :disabled="form.protected" :placeholder="t('ui.m5aaa131e1c')" :color="formErrors.username ? 'error' : undefined" />
            <p v-if="formErrors.username" class="text-xs text-error mt-1">{{ formErrors.username }}</p>
          </div>
          <div>
            <UInput type="password" v-model="form.password" :placeholder="isEditing ? t('ui.mc6d89967c5') : t('ui.ma621ab606d')" :color="formErrors.password ? 'error' : undefined" />
            <p v-if="formErrors.password" class="text-xs text-error mt-1">{{ formErrors.password }}</p>
          </div>
          <USelect
            v-model="form.role"
            :disabled="form.protected"
            :items="roleItems"
            value-key="value"
            label-key="label"
          />
          <UInput v-model="form.contactName" :placeholder="t('ui.m3303b56982')" />
          <UInput v-model="form.phone" :placeholder="t('ui.md698182914')" />
          <div>
            <UInput v-model="form.email" :placeholder="t('ui.m73075237fd')" :color="formErrors.email ? 'error' : undefined" />
            <p v-if="formErrors.email" class="text-xs text-error mt-1">{{ formErrors.email }}</p>
          </div>
          <div class="flex gap-2 md:col-span-2">
            <UButton type="submit" color="primary" :loading="savingUser" :disabled="savingUser">{{ isEditing ? t('ui.ma3030bf8f1') : t('ui.mebabc83b68') }}</UButton>
            <UButton type="button" variant="ghost" @click="resetForm">{{ t('ui.mcb5d682bac') }}</UButton>
          </div>
        </UForm>

        <div class="overflow-x-auto mt-4">
          <UTable :columns="userColumns" :data="users">
            <template #actions-cell="{ row }">
              <div class="flex gap-2">
                <UButton size="sm" variant="ghost" icon="i-lucide-pencil" @click="editUser(row.original)">{{ t('ui.m0518365699') }}</UButton>
                <UButton size="sm" variant="outline" color="error" icon="i-lucide-trash-2" :disabled="row.original.username === 'admin'" @click="confirmDelete(row.original)">{{ t('ui.m2f9daa8289') }}</UButton>
              </div>
            </template>
          </UTable>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <h2 class="text-xl font-bold flex items-center gap-2">
            <UIcon name="i-lucide-file-text" class="w-5 h-5" />
            {{ t('ui.mdff8b1cd27') }}
          </h2>
        </template>
        <div class="flex flex-wrap gap-3 items-end mb-4">
          <UInput v-model="printFilters.username" :placeholder="t('ui.m1a3f0617d6')" />
          <UInput :aria-label="t('accessibility.start_date')" type="date" v-model="printFilters.start" />
          <UInput :aria-label="t('accessibility.end_date')" type="date" v-model="printFilters.end" />
          <UButton variant="outline" @click="loadPrintRecords" icon="i-lucide-search">{{ t('ui.mbcd6771e08') }}</UButton>
        </div>
        <div class="overflow-x-auto">
          <UTable :columns="printColumns" :data="printRecords">
            <template #download-cell="{ row }">
              <UButton size="xs" variant="ghost" icon="i-lucide-download" @click="downloadFile(row.original.id)">{{ t('ui.m4673a23061') }}</UButton>
            </template>
          </UTable>
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <h2 class="text-xl font-bold flex items-center gap-2">
          <UIcon name="i-lucide-settings" class="w-5 h-5" />
          {{ t('ui.m68ea5dd4d7') }}
        </h2>
      </template>
      <div class="grid grid-cols-1 md:grid-cols-4 gap-3 items-end">
        <div>
          <label class="block text-sm font-medium mb-1">{{ t('ui.m6fa05bc79a') }}</label>
          <UInput type="number" step="1" v-model="settings.retentionDays" :placeholder="t('ui.m5f37c4ed11')" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">{{ t('ui.m7465a482e1') }}</label>
          <UInput type="number" step="1" min="0" v-model="settings.maxPagesPerJob" :placeholder="t('ui.m9f24b520d4')" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">{{ t('ui.ma231872e9a') }}</label>
          <UInput type="number" step="1" min="0" v-model="settings.maxUploadMB" :placeholder="t('ui.m9f24b520d4')" />
        </div>
        <div>
          <label class="flex items-center gap-2 cursor-pointer h-9">
            <UCheckbox v-model="settings.saveHistory" />
            <span class="text-sm">{{ t('ui.m633f94147e') }}</span>
          </label>
        </div>
        <div class="md:col-span-4">
          <label class="flex items-center gap-2 cursor-pointer">
            <UCheckbox v-model="settings.guestMode" />
            <span class="text-sm">{{ t('ui.m77e85b4c33') }}</span>
          </label>
          <p class="text-xs text-muted mt-1 ml-6">{{ t('ui.m0b377bf845') }}</p>
        </div>
        <div class="flex items-end gap-2 md:col-span-4">
          <UButton color="primary" @click="saveSettings" icon="i-lucide-save" :loading="savingSettings" :disabled="savingSettings">{{ t('ui.mc8550237ba') }}</UButton>
          <UButton variant="outline" @click="showCleanupConfirm = true" icon="i-lucide-trash-2" :loading="cleaningUp" :disabled="cleaningUp">{{ t('ui.me66632c798') }}</UButton>
        </div>
      </div>
      <div class="text-sm text-muted mt-2">{{ t('ui.m12953eab33') }}</div>

      <div class="mt-4 pt-4 border-t border-default">
        <label class="block text-sm font-medium mb-1">{{ t('ui.m10ededed56') }}</label>
        <UTextarea
          v-model="settings.customCss"
          :rows="6"
          :placeholder="t('ui.m63b011e21d')"
          class="w-full font-mono text-xs"
          :maxlength="32768"
        />
        <div class="text-xs text-muted mt-1">{{ t('ui.mc8340f1492') }}</div>
      </div>
    </UCard>

    <UModal v-model:open="showDeleteModal">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">{{ t('ui.ma3ea3c17b4') }}</h3>
          <p>{{ t('ui.m11f307ecd4') }} <strong>{{ pendingDeleteUser?.username }}</strong> {{ t('ui.m9d45d89439') }}</p>
          <p class="text-sm text-muted">{{ t('ui.m438b9267f6') }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showDeleteModal = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton color="error" :loading="!!deletingUserId" @click="executeDelete">{{ t('ui.ma3ea3c17b4') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="showCleanupConfirm">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">{{ t('ui.m7ba2d6728f') }}</h3>
          <p>{{ t('ui.m08e6da0892') }}<strong>{{ t('ui.m8b3741ab6d') }}</strong>{{ t('ui.md681da6ee2') }}</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showCleanupConfirm = false">{{ t('ui.m2cd0f3be87') }}</UButton>
            <UButton color="error" :loading="cleaningUp" @click="triggerCleanup">{{ t('ui.m7ba2d6728f') }}</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { t } from '../i18n.js'

import { ref, computed, onMounted } from 'vue'
import { getCSRF, readError } from '../utils/api'

const toast = useToast()
const emit = defineEmits(['logout'])

const users = ref([])
const form = ref({
  id: null,
  username: '',
  password: '',
  role: 'user',
  protected: false,
  contactName: '',
  phone: '',
  email: ''
})
const printFilters = ref({ username: '', start: '', end: '' })
const printRecords = ref([])
const settings = ref({ retentionDays: '', saveHistory: true, maxPagesPerJob: '0', maxUploadMB: '0', customCss: '', guestMode: false })
const showCleanupConfirm = ref(false)

const savingUser = ref(false)
const savingSettings = ref(false)
const cleaningUp = ref(false)
const deletingUserId = ref(null)
const pendingDeleteUser = ref(null)
const showDeleteModal = ref(false)
const formErrors = ref({})

const isEditing = computed(() => !!form.value.id)

const roleItems = [
  { get label() { return t('ui.mf6a2faaac2') }, value: 'user' },
  { get label() { return t('ui.me19796712f') }, value: 'admin' }
]

const userColumns = [
  { accessorKey: 'id', header: 'ID' },
  { accessorKey: 'username', get header() { return t('ui.m5aaa131e1c') } },
  { accessorKey: 'role', get header() { return t('ui.mc47b54e84e') } },
  { accessorKey: 'contactName', get header() { return t('ui.m3303b56982') } },
  { accessorKey: 'phone', get header() { return t('ui.mbf8b5d378f') } },
  { accessorKey: 'email', get header() { return t('ui.m73075237fd') } },
  { id: 'actions', get header() { return t('ui.med31fbb483') } }
]

const printColumns = [
  { accessorKey: 'createdAt', get header() { return t('ui.m8b6ff49851') } },
  { accessorKey: 'username', get header() { return t('ui.m0d0e1a86b3') } },
  { accessorKey: 'filename', get header() { return t('ui.m39932f24fe') } },
  { accessorKey: 'pages', get header() { return t('ui.m560630ba0c') } },
  { accessorKey: 'status', get header() { return t('ui.m6320b4a872') } },
  { id: 'download', get header() { return t('ui.m4673a23061') } }
]

function validateForm() {
  formErrors.value = {}
  if (!form.value.username.trim()) {
    formErrors.value.username = t('ui.m390ccdec9f')
  }
  if (!isEditing.value && !form.value.password) {
    formErrors.value.password = t('ui.m0e7d68afbf')
  }
  if (form.value.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.value.email)) {
    formErrors.value.email = t('ui.mf248e8ed36')
  }
  return Object.keys(formErrors.value).length === 0
}

function resetForm() {
  form.value = {
    id: null,
    username: '',
    password: '',
    role: 'user',
    protected: false,
    contactName: '',
    phone: '',
    email: ''
  }
  formErrors.value = {}
}

function editUser(user) {
  form.value = {
    id: user.id,
    username: user.username,
    password: '',
    role: user.role,
    protected: user.username === 'admin',
    contactName: user.contactName || '',
    phone: user.phone || '',
    email: user.email || ''
  }
  formErrors.value = {}
}

async function loadUsers() {
  const resp = await fetch('/api/admin/users', { credentials: 'include' })
  if (!resp.ok) {
    if (resp.status === 401) emit('logout')
    return
  }
  users.value = await resp.json()
}

async function saveUser() {
  if (!validateForm()) return
  savingUser.value = true
  try {
    const payload = {
      username: form.value.username,
      password: form.value.password,
      role: form.value.role,
      contactName: form.value.contactName,
      phone: form.value.phone,
      email: form.value.email
    }
    const url = isEditing.value ? `/api/admin/users/${form.value.id}` : '/api/admin/users'
    const method = isEditing.value ? 'PUT' : 'POST'
    const resp = await fetch(url, {
      method,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCSRF()
      },
      body: JSON.stringify(payload)
    })
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.m6309a3bb5b'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      if (resp.status === 401) emit('logout')
      return
    }
    toast.add({ title: isEditing.value ? t('ui.m7c0d266486') : t('ui.m1ab62884f4'), description: t('ui.me69af36504', { p0: (form.value.username) }), color: 'success', icon: 'i-lucide-check-circle' })
    await loadUsers()
    resetForm()
  } finally {
    savingUser.value = false
  }
}

function confirmDelete(user) {
  pendingDeleteUser.value = user
  showDeleteModal.value = true
}

async function executeDelete() {
  const user = pendingDeleteUser.value
  if (!user) return
  deletingUserId.value = user.id
  try {
    const resp = await fetch(`/api/admin/users/${user.id}`, {
      method: 'DELETE',
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.mc228558cf2'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      if (resp.status === 401) emit('logout')
      return
    }
    toast.add({ title: t('ui.m5223f91b96'), description: t('ui.me30b62223c', { p0: (user.username) }), color: 'success', icon: 'i-lucide-check-circle' })
    await loadUsers()
  } finally {
    deletingUserId.value = null
    showDeleteModal.value = false
    pendingDeleteUser.value = null
  }
}

function downloadFile(id) {
  window.open(`/api/print-records/${id}/file`, '_blank')
}

async function loadPrintRecords() {
  const params = new URLSearchParams()
  if (printFilters.value.username) params.set('username', printFilters.value.username)
  if (printFilters.value.start) params.set('start', printFilters.value.start)
  if (printFilters.value.end) params.set('end', printFilters.value.end)
  const resp = await fetch(`/api/admin/print-records?${params.toString()}`, { credentials: 'include' })
  if (!resp.ok) {
    if (resp.status === 401) emit('logout')
    return
  }
  printRecords.value = await resp.json()
}

async function loadSettings() {
  const resp = await fetch('/api/admin/settings', { credentials: 'include' })
  if (!resp.ok) {
    if (resp.status === 401) emit('logout')
    return
  }
  const data = await resp.json()
  settings.value.retentionDays = String(data.retentionDays || 0)
  settings.value.saveHistory = data.saveHistory !== false
  settings.value.maxPagesPerJob = String(data.maxPagesPerJob || 0)
  // 后端以字节存储，前端 UI 用 MB 展示（1 MB = 1024*1024 B）。
  const bytes = Number(data.maxUploadBytes || 0)
  settings.value.maxUploadMB = String(bytes > 0 ? Math.round(bytes / (1024 * 1024)) : 0)
  settings.value.customCss = typeof data.customCss === 'string' ? data.customCss : ''
  settings.value.guestMode = data.guestMode === true
}

async function triggerCleanup() {
  showCleanupConfirm.value = false
  cleaningUp.value = true
  try {
    const resp = await fetch('/api/admin/cleanup', {
      method: 'POST',
      credentials: 'include',
      headers: { 'X-CSRF-Token': getCSRF() }
    })
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.m09e7b98edc'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      if (resp.status === 401) emit('logout')
      return
    }
    const data = await resp.json()
    const count = data.deleted || 0
    toast.add({
      title: t('ui.m6d256e25a3'),
      description: count > 0 ? t('ui.mf695b1e02a', { p0: (count) }) : t('ui.m972db39454'),
      color: 'success',
      icon: 'i-lucide-check-circle'
    })
    await loadPrintRecords()
  } finally {
    cleaningUp.value = false
  }
}

async function saveSettings() {
  savingSettings.value = true
  try {
    const maxPages = Math.max(0, parseInt(settings.value.maxPagesPerJob || '0', 10) || 0)
    const maxMB = Math.max(0, parseInt(settings.value.maxUploadMB || '0', 10) || 0)
    const payload = {
      retentionDays: parseInt(settings.value.retentionDays || '0', 10),
      saveHistory: settings.value.saveHistory,
      maxPagesPerJob: maxPages,
      maxUploadBytes: maxMB * 1024 * 1024,
      customCss: settings.value.customCss || '',
      guestMode: settings.value.guestMode === true
    }
    const resp = await fetch('/api/admin/settings', {
      method: 'PUT',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCSRF()
      },
      body: JSON.stringify(payload)
    })
    if (!resp.ok) {
      const msg = await readError(resp)
      toast.add({ title: t('ui.m6309a3bb5b'), description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      if (resp.status === 401) emit('logout')
      return
    }
    toast.add({ title: t('ui.me05ddf992c'), get description() { return t('ui.m61e936f250') }, color: 'success', icon: 'i-lucide-check-circle' })
    await loadSettings()
    // 立即把新的自定义 CSS 应用到当前页面，避免管理员必须手动 reload（Issue #57）。
    const el = document.getElementById('app-custom-css')
    if (el || settings.value.customCss) {
      const style = el || Object.assign(document.createElement('style'), { id: 'app-custom-css' })
      if (!el) document.head.appendChild(style)
      style.textContent = settings.value.customCss || ''
    }
  } finally {
    savingSettings.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadUsers(), loadPrintRecords(), loadSettings()])
})
</script>
