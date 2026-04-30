<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import {
  Clipboard,
  KeyRound,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Server,
  Shield,
  ShieldCheck,
  Trash2,
  UserRound
} from 'lucide-vue-next'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
  NSpin,
  NSwitch,
  NTable,
  useDialog,
  useMessage
} from 'naive-ui'
import { addUser, copyText, getLocalIPAddresses, removeUser, resetUserPassword, setAuthEnabled } from '../backend/api'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'
import { friendlyError } from '../utils/errors'

const config = useConfigStore()
const server = useServerStore()
const message = useMessage()
const dialog = useDialog()

const busy = ref(false)
const showAdd = ref(false)
const showReset = ref(false)
const showCredentialInfo = ref(false)
const targetUser = ref('')
const username = ref('')
const password = ref('')
const credentialInfo = ref('')
const confirmCountdown = ref(0)
const sessionPasswords = reactive<Record<string, string>>({})
let confirmTimer: number | undefined

const users = computed(() => config.draft?.auth.users ?? [])
const authEnabled = computed({
  get: () => config.draft?.auth.enabled ?? false,
  set: async (value: boolean) => {
    await toggleAuth(value)
  }
})

function maskHash(hash: string) {
  if (!hash) return '-'
  return `${hash.slice(0, 7)}...${hash.slice(-6)}`
}

function basicToken(name: string, plainPassword: string) {
  return btoa(unescape(encodeURIComponent(`${name}:${plainPassword}`)))
}

function proxyAuthorization(name: string) {
  const plainPassword = sessionPasswords[name]
  if (!plainPassword) return ''
  return `Basic ${basicToken(name, plainPassword)}`
}

function maskedProxyAuthorization(name: string) {
  const value = proxyAuthorization(name)
  if (!value) return 'Basic ******'
  const token = value.replace('Basic ', '')
  return `Basic ${token.slice(0, 8)}...${token.slice(-6)}`
}

async function localIPText() {
  try {
    const ips = await getLocalIPAddresses()
    if (ips.length > 0) return ips.join(' / ')
  } catch {
    // Ignore adapter enumeration failures here.
  }
  return '未检测到网卡 IP'
}

function credentialPayload(name: string, plainPassword: string, localIPs: string) {
  const socksPort = config.draft?.server.socks5.port ?? ''
  const httpPort = config.draft?.server.http.port ?? ''
  const proxyAuth = `Basic ${basicToken(name, plainPassword)}`
  return [
    '===============',
    'GoProxy 连接信息',
    '===============',
    `当前 IP：${localIPs}`,
    '端口：',
    `SOCKS5：${socksPort}`,
    `HTTPS：${httpPort}`,
    'SOCKS5 连接校验信息：',
    `用户名：${name}`,
    `密码：${plainPassword}`,
    'HTTPS 连接校验信息：',
    `Proxy-Authorization：${proxyAuth}`
  ].join('\n')
}

function resetForm() {
  username.value = ''
  password.value = ''
  targetUser.value = ''
}

function startConfirmCountdown() {
  window.clearInterval(confirmTimer)
  confirmCountdown.value = 3
  confirmTimer = window.setInterval(() => {
    confirmCountdown.value -= 1
    if (confirmCountdown.value <= 0) {
      window.clearInterval(confirmTimer)
    }
  }, 1000)
}

function closeCredentialInfo() {
  if (confirmCountdown.value > 0) {
    message.warning('请先复制并确认保存，3 秒后才能关闭。')
    return
  }
  showCredentialInfo.value = false
}

async function reloadConfig() {
  await config.load()
  await server.refresh()
}

async function copyValue(value: string, successText: string) {
  if (!value) {
    message.warning('旧用户无法读取明文密码，请先重置密码后再复制。')
    return
  }
  await copyText(value)
  message.success(successText)
}

async function toggleAuth(enabled: boolean) {
  busy.value = true
  try {
    await setAuthEnabled(enabled)
    await reloadConfig()
    message.success(enabled ? '认证已开启' : '认证已关闭')
  } catch (err) {
    message.error(friendlyError(err))
  } finally {
    busy.value = false
  }
}

async function createUser() {
  const name = username.value.trim()
  const plainPassword = password.value
  if (!name || !plainPassword) {
    message.warning('请填写用户名和密码')
    return
  }
  busy.value = true
  try {
    await addUser(name, plainPassword)
    sessionPasswords[name] = plainPassword
    credentialInfo.value = credentialPayload(name, plainPassword, await localIPText())
    await reloadConfig()
    showAdd.value = false
    showCredentialInfo.value = true
    startConfirmCountdown()
    resetForm()
    message.success('用户已新增')
  } catch (err) {
    message.error(friendlyError(err))
  } finally {
    busy.value = false
  }
}

function confirmRemove(name: string) {
  dialog.warning({
    title: '删除用户',
    content: `确认删除用户 ${name}？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      busy.value = true
      try {
        await removeUser(name)
        delete sessionPasswords[name]
        await reloadConfig()
        message.success('用户已删除')
      } catch (err) {
        message.error(friendlyError(err))
      } finally {
        busy.value = false
      }
    }
  })
}

function openReset(name: string) {
  targetUser.value = name
  password.value = ''
  showReset.value = true
}

async function submitReset() {
  const plainPassword = password.value
  if (!targetUser.value || !plainPassword) {
    message.warning('请填写新密码')
    return
  }
  busy.value = true
  try {
    await resetUserPassword(targetUser.value, plainPassword)
    sessionPasswords[targetUser.value] = plainPassword
    credentialInfo.value = credentialPayload(targetUser.value, plainPassword, await localIPText())
    await reloadConfig()
    showReset.value = false
    showCredentialInfo.value = true
    startConfirmCountdown()
    resetForm()
    message.success('密码已重置')
  } catch (err) {
    message.error(friendlyError(err))
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="auth-unified-page">
    <div class="page-shell">
      <div class="page-header">
        <div class="page-header-main">
          <div class="page-header-icon">
            <NIcon :component="Shield" />
          </div>
          <div>
            <h2 class="page-title">认证管理</h2>
            <p class="page-subtitle">管理代理访问控制、用户凭据和认证状态</p>
          </div>
        </div>
        <div class="page-header-actions">
          <NButton secondary :loading="config.loading" @click="reloadConfig">
            <template #icon>
              <NIcon :component="RefreshCw" />
            </template>
            刷新
          </NButton>
          <NButton type="primary" @click="showAdd = true">
            <template #icon>
              <NIcon :component="Plus" />
            </template>
            新增用户
          </NButton>
        </div>
      </div>

      <NAlert v-if="authEnabled && users.length === 0" type="warning" class="page-alert">
        开启认证前至少需要一个用户。
      </NAlert>
      <NAlert v-if="!authEnabled" type="warning" class="page-alert">
        当前未开启认证，请确认监听地址不会暴露到不可信网络。
      </NAlert>

      <NSpin :show="config.loading">
        <template v-if="config.draft">
          <div>
            <section class="config-card">
              <div class="card-head">
                <div class="card-title-wrap">
                  <div class="card-icon access">
                    <NIcon :component="ShieldCheck" />
                  </div>
                  <div>
                    <div class="card-title">访问控制</div>
                    <div class="card-subtitle">BASIC / RFC1929</div>
                  </div>
                </div>
              </div>

              <div class="card-content">
                <div class="surface-block">
                  <div class="field-row field-row-switch">
                    <div class="field-label-group">
                      <div class="field-title">代理认证</div>
                      <div class="field-desc">SOCKS5 使用用户名和密码，HTTP CONNECT 使用 Proxy-Authorization Basic</div>
                    </div>
                    <NSwitch v-model:value="authEnabled" :loading="busy" />
                  </div>
                </div>
              </div>
            </section>

            <section class="config-card config-card-spaced">
              <div class="card-head">
                <div class="card-title-wrap">
                  <div class="card-icon users">
                    <NIcon :component="UserRound" />
                  </div>
                  <div>
                    <div class="card-title">用户列表</div>
                    <div class="card-subtitle">{{ users.length }} USERS</div>
                  </div>
                </div>
              </div>

              <div class="card-content">
                <div class="table-shell">
                  <NTable :bordered="false" :single-line="false" class="auth-table">
                    <thead>
                      <tr>
                        <th>用户名</th>
                        <th>密码 Hash</th>
                        <th>Proxy-Authorization</th>
                        <th>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-if="users.length === 0">
                        <td colspan="4" class="table-empty">暂无认证用户</td>
                      </tr>
                      <tr v-for="user in users" :key="user.username">
                        <td class="mono-cell">{{ user.username }}</td>
                        <td class="mono-cell">{{ maskHash(user.password) }}</td>
                        <td class="mono-cell">{{ maskedProxyAuthorization(user.username) }}</td>
                        <td>
                          <div class="row-actions">
                            <NButton size="small" secondary @click="openReset(user.username)">
                              <template #icon>
                                <NIcon :component="KeyRound" />
                              </template>
                              重置密码
                            </NButton>
                            <NButton size="small" type="error" secondary @click="confirmRemove(user.username)">
                              <template #icon>
                                <NIcon :component="Trash2" />
                              </template>
                              删除
                            </NButton>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </NTable>
                </div>
              </div>
            </section>
          </div>
        </template>
      </NSpin>
    </div>

    <NModal
      v-model:show="showAdd"
      preset="dialog"
      title="新增用户"
      positive-text="保存"
      negative-text="取消"
      :loading="busy"
      @positive-click="createUser"
      @after-leave="resetForm"
    >
      <NForm label-placement="top">
        <NFormItem label="用户名">
          <NInput v-model:value="username" placeholder="admin" />
        </NFormItem>
        <NFormItem label="密码">
          <NInput v-model:value="password" type="password" show-password-on="click" />
        </NFormItem>
      </NForm>
    </NModal>

    <NModal
      :show="showCredentialInfo"
      :mask-closable="false"
      :close-on-esc="false"
      :closable="confirmCountdown <= 0"
      :show-icon="false"
      preset="dialog"
      title="连接校验信息"
      @update:show="(value) => { if (!value) closeCredentialInfo() }"
    >
      <NAlert type="warning" class="page-alert">
        请立即复制并妥善保存，关闭后无法查看密码。
      </NAlert>
      <pre class="credential-box">{{ credentialInfo }}</pre>
      <template #action>
        <div class="page-header-actions">
          <NButton secondary @click="copyValue(credentialInfo, '连接信息已复制')">
            <template #icon>
              <NIcon :component="Clipboard" />
            </template>
            一键复制
          </NButton>
          <NButton type="primary" :disabled="confirmCountdown > 0" @click="closeCredentialInfo">
            {{ confirmCountdown > 0 ? `${confirmCountdown}s 后确认` : '确认' }}
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="showReset"
      preset="dialog"
      title="重置密码"
      positive-text="保存"
      negative-text="取消"
      :loading="busy"
      @positive-click="submitReset"
      @after-leave="resetForm"
    >
      <NForm label-placement="top">
        <NFormItem label="用户">
          <NInput :value="targetUser" readonly>
            <template #prefix>
              <NIcon :component="ShieldCheck" />
            </template>
          </NInput>
        </NFormItem>
        <NFormItem label="新密码">
          <NInput v-model:value="password" type="password" show-password-on="click" />
        </NFormItem>
      </NForm>
    </NModal>
  </section>
</template>

<style scoped>
.auth-unified-page {
  width: 100%;
}

.page-shell {
  max-width: 920px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.page-header-main {
  display: flex;
  align-items: center;
  gap: 14px;
}

.page-header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.page-header-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--panel) 86%, var(--fg-soft) 14%);
  color: var(--fg);
  font-size: 18px;
  border: 1px solid var(--border);
}

.page-title {
  margin: 0;
  font-size: 28px;
  line-height: 1.15;
  font-weight: 600;
  color: var(--fg);
}

.page-subtitle {
  margin: 6px 0 0;
  font-size: 14px;
  color: var(--fg-soft);
}

.page-alert {
  margin: 0;
}

.config-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.config-card-spaced {
  margin-top: 24px;
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 24px 24px 0;
}

.card-title-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.card-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
}

.card-icon.access {
  color: #059669;
  background: rgba(16, 185, 129, 0.12);
}

.card-icon.users {
  color: #0284c7;
  background: rgba(14, 165, 233, 0.12);
}

.card-title {
  font-size: 16px;
  line-height: 1;
  font-weight: 600;
  color: var(--fg);
}

.card-subtitle {
  margin-top: 6px;
  font-size: 13px;
  color: var(--fg-soft);
}

.card-content {
  padding: 24px;
}

.surface-block,
.table-shell {
  border-radius: 10px;
  border: 1px solid var(--border);
  background: color-mix(in srgb, var(--panel) 90%, var(--fg-soft) 10%);
}

.surface-block {
  padding: 18px 16px;
}

.field-row-switch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.field-label-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fg);
}

.field-desc {
  font-size: 13px;
  color: var(--fg-soft);
}

.table-shell {
  overflow: hidden;
}

.auth-table :deep(th) {
  color: var(--fg-soft);
  font-size: 12px;
  font-weight: 600;
}

.auth-table :deep(td) {
  color: var(--fg);
}

.mono-cell {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
}

.table-empty {
  text-align: center;
  color: var(--fg-soft);
  padding: 28px 16px !important;
}

.row-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.credential-box {
  margin: 14px 0 0;
  padding: 14px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--panel) 86%, var(--fg-soft) 14%);
  border: 1px solid var(--border);
  color: var(--fg);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.auth-unified-page :deep(.n-input),
.auth-unified-page :deep(.n-base-selection) {
  --n-border-radius: 8px !important;
}

@media (max-width: 720px) {
  .page-header,
  .page-header-main,
  .field-row-switch {
    flex-direction: column;
    align-items: flex-start;
  }

  .page-header-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
