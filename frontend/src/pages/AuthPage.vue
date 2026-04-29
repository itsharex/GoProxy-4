<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { Clipboard, KeyRound, Plus, RefreshCw, ShieldCheck, Trash2 } from 'lucide-vue-next'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NIcon,
  NInput,
  NModal,
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
    if (ips.length > 0) return ips.join('、')
  } catch {
    // Keep the dialog usable even if adapter enumeration fails.
  }
  return '未检测到网卡 IP'
}

function credentialPayload(name: string, plainPassword: string, localIPs: string) {
  const socksPort = config.draft?.server.socks5.port ?? ''
  const httpPort = config.draft?.server.http.port ?? ''
  const proxyAuth = `Basic ${basicToken(name, plainPassword)}`
  return [
    '===============',
    'GoProxy连接信息',
    '===============',
    `当前IP：${localIPs}`,
    '端口：',
    `Socks5：${socksPort}`,
    `HTTPS：${httpPort}`,
    'Socks5连接校验信息：',
    `用户名：${name}`,
    `密    码：${plainPassword}`,
    'HTTPS连接校验信息：',
    `Proxy-Authorization：${proxyAuth}`
  ].join('\n')
}

function copyPayload(name: string) {
  const plainPassword = sessionPasswords[name]
  if (!plainPassword) return ''
  return credentialPayload(name, plainPassword, '请以新增或重置弹窗中的网卡 IP 为准')
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
  <section class="auth-page">
    <div class="section-actions">
      <div>
        <span class="section-kicker">AUTH</span>
        <h2>认证管理</h2>
      </div>
      <div class="header-actions">
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

    <section class="panel form-panel">
      <div class="panel-head">
        <h3>访问控制</h3>
        <span class="tag">BASIC / RFC1929</span>
      </div>
      <div class="auth-toggle-row">
        <div>
          <strong>代理认证</strong>
          <span>SOCKS5 使用用户名和密码；HTTP CONNECT 使用 Proxy-Authorization Basic。</span>
        </div>
        <NSwitch v-model:value="authEnabled" :loading="busy" />
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h3>用户列表</h3>
        <span class="tag">{{ users.length }} USERS</span>
      </div>
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
    </section>

    <NModal v-model:show="showAdd" preset="dialog" title="新增用户" positive-text="保存" negative-text="取消" :loading="busy" @positive-click="createUser" @after-leave="resetForm">
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
        <div class="header-actions">
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

    <NModal v-model:show="showReset" preset="dialog" title="重置密码" positive-text="保存" negative-text="取消" :loading="busy" @positive-click="submitReset" @after-leave="resetForm">
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
