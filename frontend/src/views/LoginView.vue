<template>
  <div class="flex items-center justify-center h-full p-3 sm:p-4 md:p-6">
    <UCard class="w-full max-w-md shadow-lg">
      <template #header>
        <h2 class="text-xl font-bold flex items-center gap-2">
          <UIcon name="i-lucide-user" class="w-5 h-5" />
          {{ t('ui.m1e2df9c307') }}
        </h2>
      </template>
      
      <!-- 错误提示 -->
      <UAlert
        v-if="error"
        icon="i-lucide-triangle-alert"
        color="error"
        variant="soft"
        :title="error"
        class="mb-6"
      />
      
      <UForm @submit="login" :state="state" class="space-y-6">
        <UFormField :label="t('ui.m1a3f0617d6')" name="username" required>
          <UInput v-model="state.username" icon="i-lucide-user" size="lg" class="w-full" />
        </UFormField>
        <UFormField :label="t('ui.ma621ab606d')" name="password" required>
          <UInput v-model="state.password" type="password" icon="i-lucide-lock" size="lg" class="w-full" />
        </UFormField>
        
        <div class="mt-6">
          <UButton 
            type="submit" 
            color="primary" 
            icon="i-lucide-log-in" 
            size="lg"
            class="w-full"
            :loading="loading"
          >
            {{ t('ui.m1e2df9c307') }}
          </UButton>
        </div>
      </UForm>
    </UCard>
  </div>
</template>

<script setup>
import { t } from '../i18n.js'

import { reactive, ref } from 'vue'

const state = reactive({
  username: '',
  password: ''
})
const error = ref('')
const loading = ref(false)

const emit = defineEmits(['login-success'])

// 后端 /api/login 的错误串是英文短语，直接展示对终端用户不友好，在此本地化。
const BACKEND_MESSAGES = {
  'invalid credentials': t('ui.ma9a6a83a7d'),
  'missing credentials': t('ui.m1793872ca1'),
  'too many attempts, please try again later': t('ui.me584fd65a0'),
  'login failed': t('ui.m29056d6d0f'),
  'session error': t('ui.m299def6bcb')
}

// 把失败响应翻译成可诊断的中文提示。
//
// 关键约定：响应体不是后端 JSON 时，绝不能兜底显示「用户名或密码错误」。这类失败说明
// 请求根本没走到 LoginHandler（被跨源防护、反向代理或网关拦掉），谎报成凭据问题会把
// 排查方向彻底带偏 —— 这正是 issue #99 里「内网能登录，反代后显示密码错误」的由来。
async function describeFailure(resp) {
  // body 只能读一次，先取文本再自行尝试解析。
  let raw = ''
  try {
    raw = await resp.text()
  } catch {
    raw = ''
  }

  let payload = null
  try {
    payload = JSON.parse(raw)
  } catch {
    payload = null
  }

  if (payload && (payload.error || payload.message)) {
    const msg = payload.error || payload.message
    return BACKEND_MESSAGES[msg] || msg
  }

  const looksLikeHTML = raw.trimStart().startsWith('<')
  const detail = looksLikeHTML
    ? t('ui.mf629084e44')
    : raw.trim().slice(0, 200)

  if (resp.status === 403) {
    return t('ui.m0650ce58ee', { p0: (detail) })
  }
  if (resp.status === 404 || resp.status === 405) {
    return t('ui.mbc7730264a', { p0: (resp.status), p1: (detail) })
  }
  if (resp.status >= 500) {
    return t('ui.m9657aed5be', { p0: (resp.status), p1: (detail) })
  }
  return t('ui.maba149aa8a', { p0: (resp.status), p1: (detail ? '：' + detail : '') })
}

async function login() {
  error.value = ''
  loading.value = true
  try {
    const resp = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: state.username, password: state.password }),
      credentials: 'include'
    })
    if (!resp.ok) {
      error.value = await describeFailure(resp)
      return
    }
    emit('login-success')
  } catch (e) {
    // fetch 本身抛异常＝请求没能完成（网络不可达、TLS 失败、被浏览器策略阻断）。
    error.value = t('ui.m511a9ffe95', { p0: (e.message) })
  } finally {
    loading.value = false
  }
}
</script>
