<template>
  <main class="max-w-2xl mx-auto p-4 sm:p-6 space-y-6">
    <h2 class="text-xl font-bold">{{ t('account.title') }}</h2>
    <UAlert v-if="error" color="error" :title="error" />
    <UAlert v-if="success" color="success" :title="success" />
    <UAlert v-if="isGuest" :title="t('account.guest_read_only')" />
    <template v-else>
      <UCard>
        <form @submit.prevent="saveLanguage" class="space-y-4">
          <UFormField :label="t('account.language')" name="language">
            <USelect v-model="language" :items="languageItems" class="w-full" :disabled="loading" />
          </UFormField>
          <UButton type="submit" :loading="saving" :disabled="loading">{{ t('account.save') }}</UButton>
        </form>
      </UCard>
      <UCard>
        <form @submit.prevent="changePassword" class="space-y-4">
          <h3 class="font-semibold">{{ t('account.change_password') }}</h3>
          <p class="text-sm text-muted">{{ t('account.password_help') }}</p>
          <UFormField :label="t('account.current_password')" name="currentPassword">
            <UInput v-model="currentPassword" type="password" autocomplete="current-password" required class="w-full" />
          </UFormField>
          <UFormField :label="t('account.new_password')" name="newPassword">
            <UInput v-model="newPassword" type="password" autocomplete="new-password" required class="w-full" />
          </UFormField>
          <UFormField :label="t('account.confirm_password')" name="confirmPassword">
            <UInput v-model="confirmPassword" type="password" autocomplete="new-password" required class="w-full" />
          </UFormField>
          <UButton type="submit" :loading="changing" :disabled="loading">{{ t('account.change_password') }}</UButton>
        </form>
      </UCard>
    </template>
  </main>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { apiFetch, readError } from '../utils/api'
import { t, setLanguage } from '../i18n'
import { validPassword } from '../utils/account'

const props = defineProps({ session: Object })
const emit = defineEmits(['logout'])
const isGuest = computed(() => props.session?.username === 'guest')
const language = ref('server-default'), currentPassword = ref(''), newPassword = ref(''), confirmPassword = ref('')
const error = ref(''), success = ref(''), loading = ref(true), saving = ref(false), changing = ref(false)
const languageItems = computed(() => [
  { value: 'server-default', label: t('account.server_default') },
  { value: 'en', label: t('account.english') },
  { value: 'zh-CN', label: t('account.chinese') }
])
const signOut = () => emit('logout')
onMounted(async () => {
  try {
    const [profile, csrf] = await Promise.all([apiFetch('/api/me', {}, signOut), apiFetch('/api/csrf', {}, signOut)])
    if (!profile.ok) throw new Error(await readError(profile))
    if (!csrf.ok) throw new Error(await readError(csrf))
    language.value = (await profile.json()).language || 'server-default'
  } catch (e) { error.value = e.message } finally { loading.value = false }
})
async function saveLanguage() {
  error.value = ''; success.value = ''; saving.value = true
  try {
    const response = await apiFetch('/api/me/preferences', { method: 'PUT', body: JSON.stringify({ language: language.value === 'server-default' ? '' : language.value }) }, signOut)
    if (!response.ok) throw new Error(await readError(response))
    setLanguage((await response.json()).effectiveLanguage)
    success.value = t('account.saved')
  } catch (e) { error.value = e.message } finally { saving.value = false }
}
async function changePassword() {
  error.value = ''; success.value = ''
  if (!validPassword(newPassword.value)) { error.value = t('account.password_length'); return }
  if (newPassword.value !== confirmPassword.value) { error.value = t('account.password_mismatch'); return }
  changing.value = true
  try {
    const response = await apiFetch('/api/me/password', { method: 'PUT', body: JSON.stringify({ currentPassword: currentPassword.value, newPassword: newPassword.value }) }, signOut)
    if (!response.ok) throw new Error(await readError(response))
    currentPassword.value = ''; newPassword.value = ''; confirmPassword.value = ''
    signOut()
  } catch (e) { error.value = e.message } finally { changing.value = false }
}
</script>
