<template>
  <UApp :locale="uiLocale">
    <div v-if="!sessionLoaded" class="flex items-center justify-center min-h-screen bg-default">
      <UIcon name="i-lucide-loader-circle" class="w-8 h-8 animate-spin text-primary" />
    </div>
    <div v-else class="grid grid-rows-[auto_1fr_auto] min-h-screen w-full bg-default">
      <header class="flex items-center justify-between px-4 sm:px-6 py-3 border-b border-default bg-default">
        <div class="flex items-center gap-3 min-w-0">
          <h1 class="text-xl font-bold shrink-0">{{ t('ui.mc3cd485558') }}</h1>
          <span v-if="session" class="text-sm text-muted truncate">{{ session.username }}</span>
        </div>
        <div class="flex items-center gap-2">
          <!-- 桌面端（sm+）：分段按钮 + 文字，一目了然 -->
          <div class="hidden sm:flex items-center gap-2">
            <!-- 导航分段容器：与主 CTA 视觉区分。
                 「打印 / 定时」对所有登录用户开放；「管理 / 驱动」仅管理员可见。 -->
            <div
              v-if="session"
              class="flex items-center gap-0.5 p-0.5 rounded-lg bg-elevated/60 border border-default"
            >
              <UButton
                :variant="route.path === '/print' ? 'soft' : 'ghost'"
                :color="route.path === '/print' ? 'primary' : 'neutral'"
                size="xs"
                icon="i-lucide-file-text"
                @click="router.push('/print')"
              >
                {{ t('ui.md7bfe7b505') }}
              </UButton>
              <UButton
                :variant="route.path === '/scheduled' ? 'soft' : 'ghost'"
                :color="route.path === '/scheduled' ? 'primary' : 'neutral'"
                size="xs"
                icon="i-lucide-calendar-clock"
                @click="router.push('/scheduled')"
              >
                {{ t('ui.m6a2b5eb8d4') }}
              </UButton>
              <UButton
                :variant="route.path === '/scan' ? 'soft' : 'ghost'"
                :color="route.path === '/scan' ? 'primary' : 'neutral'"
                size="xs"
                icon="i-lucide-scan-line"
                @click="router.push('/scan')"
              >
                {{ t('ui.mc2a291b64f') }}
              </UButton>
              <UButton
                v-if="canManageKeys"
                :variant="route.path === '/api-keys' ? 'soft' : 'ghost'"
                :color="route.path === '/api-keys' ? 'primary' : 'neutral'"
                size="xs"
                icon="i-lucide-key-round"
                @click="router.push('/api-keys')"
              >
                {{ t('ui.mf67bca8f42') }}
              </UButton>
              <UButton
                v-if="isAdmin"
                :variant="route.path === '/admin' ? 'soft' : 'ghost'"
                :color="route.path === '/admin' ? 'primary' : 'neutral'"
                size="xs"
                icon="i-lucide-settings"
                @click="router.push('/admin')"
              >
                {{ t('ui.mbb6d995724') }}
              </UButton>
              <UButton
                v-if="isAdmin"
                :variant="route.path === '/drivers' ? 'soft' : 'ghost'"
                :color="route.path === '/drivers' ? 'primary' : 'neutral'"
                size="xs"
                icon="i-lucide-puzzle"
                @click="router.push('/drivers')"
              >
                {{ t('ui.m8d72392eda') }}
              </UButton>
            </div>
            <UButton v-if="canManageKeys" variant="ghost" color="neutral" size="xs" icon="i-lucide-user-round" @click="router.push('/account')">
              {{ t('account.title') }}
            </UButton>
            <UButton
              v-if="session"
              variant="ghost"
              color="neutral"
              size="xs"
              icon="i-lucide-log-out"
              @click="logout"
            >
              {{ t('ui.m057f31bc16') }}
            </UButton>
          </div>
          <!-- 移动端（<sm）：折叠为汉堡菜单，图标+文字，易点易读 -->
          <UDropdownMenu
            v-if="session"
            :items="menuItems"
            :content="{ align: 'end' }"
            class="sm:hidden"
          >
            <UButton variant="ghost" color="neutral" size="sm" icon="i-lucide-menu" :aria-label="t('account.menu')" square />
          </UDropdownMenu>
        </div>
      </header>
      <div class="overflow-auto relative">
        <router-view :session="session" @login-success="onLogin" @logout="onLogout" />
      </div>
      <footer class="px-4 sm:px-6 py-3 border-t border-default bg-default text-sm text-muted text-center space-y-1">
        <div class="flex items-center justify-center gap-x-3 gap-y-1 flex-wrap">
          <a href="https://gitlab.com/Alcedema/cups-web" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">{{ t('footer.project_name') }}</a>
          <template v-if="appVersion">
            <span aria-hidden="true" class="text-default/40">·</span>
            <span class="font-mono text-xs" :title="`cups-web ${appVersion}`">{{ appVersion }}</span>
          </template>
          <span aria-hidden="true" class="text-default/40">·</span>
          <a href="https://gitlab.com/Alcedema/cups-web/-/issues" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">{{ t('footer.report_issue') }}</a>
        </div>
        <div class="flex items-center justify-center gap-x-3 gap-y-1 flex-wrap text-xs">
          <span>
            {{ t('footer.based_on') }}
            <a href="https://github.com/hanxi/cups-web" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">{{ t('footer.upstream_name') }}</a>
          </span>
          <span aria-hidden="true" class="text-default/40">·</span>
          <a href="/LICENSE.txt" target="_blank" rel="noopener noreferrer" class="text-primary hover:underline">{{ t('footer.licence') }}</a>
        </div>
      </footer>
    </div>
  </UApp>
</template>

<script setup>
import { t } from './i18n.js'
import { uiLocale, setLanguage, setServerLanguage } from './i18n.js'

import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { clearSessionCache, updateSessionCache } from './router'

const router = useRouter()
const route = useRoute()

const session = ref(null)
const sessionLoaded = ref(false)
// 二进制版本号：首次挂载时拉一次 /api/version（公开接口，不要求登录），
// 失败时保持空字符串，footer 上的版本号节点会被 v-if 隐藏，不影响布局。
const appVersion = ref('')

const isAdmin = computed(() => session.value?.role === 'admin')
// 访客模式（Issue #55）签发的 guest 账号不允许管理 API 密钥（Issue #113），
// 避免访客通过 UI 生成 key 后带出容器；页面本身有二次校验，这里只是隐入口。
const canManageKeys = computed(() => session.value && session.value.username !== 'guest')

// 移动端汉堡菜单项：导航项（仅 admin）与登出分成两组，组间自动加分隔线
const menuItems = computed(() => {
  const nav = []
  if (session.value) {
    nav.push({ get label() { return t('ui.md7bfe7b505') }, icon: 'i-lucide-file-text', onSelect: () => router.push('/print') })
    nav.push({ get label() { return t('ui.m6a2b5eb8d4') }, icon: 'i-lucide-calendar-clock', onSelect: () => router.push('/scheduled') })
    nav.push({ get label() { return t('ui.mc2a291b64f') }, icon: 'i-lucide-scan-line', onSelect: () => router.push('/scan') })
  }
  if (canManageKeys.value) {
    nav.push({ label: t('account.title'), icon: 'i-lucide-user-round', onSelect: () => router.push('/account') })
    nav.push({ get label() { return t('ui.mf67bca8f42') }, icon: 'i-lucide-key-round', onSelect: () => router.push('/api-keys') })
  }
  if (isAdmin.value) {
    nav.push({ get label() { return t('ui.mbb6d995724') }, icon: 'i-lucide-settings', onSelect: () => router.push('/admin') })
    nav.push({ get label() { return t('ui.m8d72392eda') }, icon: 'i-lucide-puzzle', onSelect: () => router.push('/drivers') })
  }
  const account = [{ get label() { return t('ui.m057f31bc16') }, icon: 'i-lucide-log-out', onSelect: () => logout() }]
  return nav.length ? [nav, account] : [account]
})

async function loadVersion() {
  try {
    const resp = await fetch('/api/version', { credentials: 'include' })
    if (resp.ok) {
      const data = await resp.json()
      if (data && typeof data.version === 'string') {
        appVersion.value = data.version
      }
    }
  } catch (e) {
    // 版本号展示属于可降级的信息，静默失败即可
  }
}

// 自定义 CSS 注入（Issue #57）：管理员在设置里贴一段 CSS，登录前也应生效
// （登录页也是应用的一部分），因此走公开接口。用 <style id="app-custom-css">
// 作为幂等的注入点，保存后可以随时刷新样式而不用 reload 整页。
async function loadCustomCSS() {
  try {
    const resp = await fetch('/api/public-settings')
    if (!resp.ok) return
    const data = await resp.json()
    setServerLanguage(data.defaultLanguage)
    if (data && typeof data.customCss === 'string') {
      applyCustomCSS(data.customCss)
    }
  } catch (e) {
    // 自定义样式属于装饰性功能，接口挂了不该阻塞主界面
  }
}

function applyCustomCSS(css) {
  let el = document.getElementById('app-custom-css')
  if (!el) {
    el = document.createElement('style')
    el.id = 'app-custom-css'
    document.head.appendChild(el)
  }
  el.textContent = css || ''
}

async function loadSession() {
  try {
    // /session also creates the shared guest session when guest mode is enabled.
    const resp = await fetch('/api/session', { credentials: 'include' })
    if (resp.ok) {
      const profile = await fetch('/api/me', { credentials: 'include' })
      if (!profile.ok) { onLogout(); return }
      const data = await profile.json()
      session.value = data
      updateSessionCache(data)
      setLanguage(data.effectiveLanguage)
      if (route.path === '/' || route.path === '/login') router.push('/print')
    } else {
      session.value = null
      router.push('/login')
    }
  } catch (e) {
    session.value = null
  } finally {
    sessionLoaded.value = true
  }
}

function onLogin() {
  loadSession()
}

function onLogout() {
  session.value = null
  setLanguage('')
  clearSessionCache()
  router.push('/login')
}

async function logout() {
  try {
    await fetch('/api/logout', { method: 'POST', credentials: 'include' })
  } catch (e) {
    // ignore errors
  }
  onLogout()
}

function detectOS() {
  if (navigator.userAgent.indexOf('Windows') !== -1) {
    document.documentElement.classList.add('is-windows')
  }
}

onMounted(async () => {
  detectOS()
  await loadCustomCSS()
  await loadSession()
  loadVersion()
})
</script>
