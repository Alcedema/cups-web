<template>
  <div class="p-3 sm:p-4 md:p-6 max-w-5xl mx-auto space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <h1 class="text-lg font-bold flex items-center gap-2">
        <UIcon name="i-lucide-key-round" class="w-5 h-5 text-primary" />
        API 密钥
      </h1>
      <div class="flex items-center gap-2">
        <UButton
          variant="ghost"
          size="xs"
          icon="i-lucide-refresh-cw"
          :loading="loading"
          @click="loadKeys"
        >刷新</UButton>
        <UButton
          size="sm"
          color="primary"
          icon="i-lucide-plus"
          @click="openCreate"
        >新建密钥</UButton>
      </div>
    </div>

    <UAlert
      color="neutral"
      variant="soft"
      icon="i-lucide-info"
      title="使用说明"
      description="密钥以持有该密钥的用户身份调用 API，可用于第三方系统（如微信机器人）直接对接。请求头带上 Authorization: Bearer <密钥> 即可，无需再走登录/CSRF。密钥仅在创建时明文返回一次，请妥善保存。"
    />

    <div v-if="loading" class="flex items-center justify-center py-10 text-muted gap-2">
      <UIcon name="i-lucide-loader-circle" class="w-5 h-5 animate-spin" />
      加载中…
    </div>
    <div v-else-if="!keys.length" class="text-center py-16 text-muted">
      <UIcon name="i-lucide-key-off" class="w-10 h-10 mx-auto mb-2 opacity-40" />
      <div>还没有 API 密钥</div>
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
                {{ isExpired(k) ? '已过期' : `到期 ${formatDate(k.expiresAt)}` }}
              </UBadge>
              <UBadge v-else color="success" variant="soft" size="sm">永不过期</UBadge>
            </div>
            <div class="text-xs text-muted space-y-0.5">
              <div>创建于 {{ formatDate(k.createdAt) }}</div>
              <div v-if="k.lastUsedAt">
                最近使用 {{ formatDate(k.lastUsedAt) }}
                <span v-if="k.lastUsedIp" class="ml-1 text-default/50">from {{ k.lastUsedIp }}</span>
              </div>
              <div v-else class="text-default/50">尚未使用</div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <UButton
              size="xs"
              variant="outline"
              color="error"
              icon="i-lucide-trash-2"
              @click="confirmDelete(k)"
            >删除</UButton>
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
            新建 API 密钥
          </h3>
          <div class="space-y-3">
            <div>
              <label class="block text-sm font-medium mb-1">备注名</label>
              <UInput v-model="createForm.name" placeholder="例如 微信机器人 / 家庭打印脚本" maxlength="64" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">有效期</label>
              <USelect
                v-model="createForm.expiresInDays"
                :items="expiryOptions"
                value-key="value"
                label-key="label"
              />
              <p class="text-xs text-muted mt-1">到期后密钥自动失效；如需长期使用可选"永不过期"，泄露时手动删除即可。</p>
            </div>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <UButton variant="ghost" @click="showCreate = false">取消</UButton>
            <UButton
              color="primary"
              :loading="creating"
              :disabled="!createForm.name.trim() || creating"
              @click="submitCreate"
            >创建</UButton>
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
            密钥已生成，仅此一次可见
          </h3>
          <p class="text-sm text-muted">
            请立即复制并妥善保存。关闭本对话框后将无法再查看完整密钥；如果遗失，请删除后重新创建。
          </p>
          <div class="p-3 rounded bg-elevated/60 border border-default">
            <div class="flex items-center gap-2">
              <code class="font-mono text-sm break-all flex-1 select-all">{{ revealedKey }}</code>
              <UButton
                size="xs"
                variant="outline"
                icon="i-lucide-copy"
                @click="copyKey"
              >复制</UButton>
            </div>
          </div>
          <div class="text-xs text-muted space-y-1">
            <div>调用示例：</div>
            <pre class="p-2 rounded bg-elevated/60 border border-default overflow-x-auto"><code>curl -H "Authorization: Bearer {{ revealedKey }}" \
     {{ origin }}/api/printers</code></pre>
          </div>
          <div class="flex justify-end">
            <UButton color="primary" @click="closeReveal">我已保存</UButton>
          </div>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="showDelete">
      <template #content>
        <div class="p-6 space-y-4">
          <h3 class="text-lg font-semibold">确认删除</h3>
          <p>确定要删除密钥 <strong>{{ pendingDelete?.name }}</strong>（{{ pendingDelete?.prefix }}）吗？</p>
          <p class="text-sm text-muted">删除后使用该密钥的请求将立即被拒绝，操作不可撤销。</p>
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showDelete = false">取消</UButton>
            <UButton color="error" :loading="deleting" @click="executeDelete">确认删除</UButton>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
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
  { label: '永不过期', value: 0 },
  { label: '7 天', value: 7 },
  { label: '30 天', value: 30 },
  { label: '90 天', value: 90 },
  { label: '一年', value: 365 }
]

function formatDate(v) {
  if (!v) return ''
  try {
    const d = new Date(v)
    if (Number.isNaN(d.getTime())) return v
    return d.toLocaleString()
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
      toast.add({ title: '加载失败', description: msg, color: 'error', icon: 'i-lucide-x-circle' })
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
      toast.add({ title: '创建失败', description: msg, color: 'error', icon: 'i-lucide-x-circle' })
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
    toast.add({ title: '已复制', color: 'success', icon: 'i-lucide-check-circle' })
  } catch {
    toast.add({ title: '复制失败', description: '请手动选择并复制', color: 'warning', icon: 'i-lucide-alert-triangle' })
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
      toast.add({ title: '删除失败', description: msg, color: 'error', icon: 'i-lucide-x-circle' })
      return
    }
    toast.add({ title: '删除成功', color: 'success', icon: 'i-lucide-check-circle' })
    await loadKeys()
  } finally {
    deleting.value = false
    showDelete.value = false
    pendingDelete.value = null
  }
}

onMounted(loadKeys)
</script>
