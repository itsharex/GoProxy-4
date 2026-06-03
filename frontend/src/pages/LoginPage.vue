<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NCard, NInput, NIcon, NModal, useMessage } from 'naive-ui'
import { LogIn, KeyRound } from 'lucide-vue-next'
import { webLogin, changePassword } from '../backend/api'

const username = ref('')
const password = ref('')
const loading = ref(false)
const message = useMessage()

const showChangePasswordModal = ref(false)
const cpOldPassword = ref('')
const cpNewPassword = ref('')
const cpConfirmPassword = ref('')
const cpLoading = ref(false)

const mustChangePwdState = ref(false)

const emit = defineEmits<{
  (e: 'login-success'): void
}>()

async function handleLogin() {
  if (!username.value.trim() || !password.value) {
    message.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const result = await webLogin(username.value.trim(), password.value)
    if (result.mustChangePwd) {
      mustChangePwdState.value = true
      showChangePasswordModal.value = true
      cpOldPassword.value = password.value
      cpNewPassword.value = ''
      cpConfirmPassword.value = ''
      message.warning('首次登录请修改密码')
    } else {
      message.success('登录成功')
      emit('login-success')
      location.hash = ''
      setTimeout(() => location.reload(), 300)
    }
  } catch (err: any) {
    message.error(err?.message || '登录失败')
  } finally {
    loading.value = false
  }
}

async function handleChangePassword() {
  if (!cpNewPassword.value || cpNewPassword.value.length < 6) {
    message.warning('新密码长度不能少于 6 位')
    return
  }
  if (cpNewPassword.value !== cpConfirmPassword.value) {
    message.warning('两次输入的新密码不一致')
    return
  }
  if (cpNewPassword.value === cpOldPassword.value) {
    message.warning('新密码不能与旧密码相同')
    return
  }
  cpLoading.value = true
  try {
    const result = await changePassword(cpOldPassword.value, cpNewPassword.value)
    message.success('密码修改成功')
    showChangePasswordModal.value = false
    mustChangePwdState.value = false
    emit('login-success')
    location.hash = ''
    setTimeout(() => location.reload(), 300)
  } catch (err: any) {
    message.error(err?.message || '密码修改失败')
  } finally {
    cpLoading.value = false
  }
}

function openChangePassword() {
  showChangePasswordModal.value = true
  cpOldPassword.value = ''
  cpNewPassword.value = ''
  cpConfirmPassword.value = ''
}

function handleCpSubmit() {
  if (!cpOldPassword.value) {
    message.warning('请输入旧密码')
    return
  }
  handleChangePassword()
}
</script>

<template>
  <div class="login-page">
    <NCard class="login-card" :bordered="false">
      <div class="login-header">
        <h1 class="login-title">GoProxy</h1>
        <p class="login-subtitle">Web 管理面板</p>
      </div>
      <form class="login-form" @submit.prevent="handleLogin">
        <div class="login-field">
          <label class="login-label">用户名</label>
          <NInput
            v-model:value="username"
            placeholder="请输入用户名"
            size="large"
            autocomplete="username"
            :disabled="loading"
          />
        </div>
        <div class="login-field">
          <label class="login-label">密码</label>
          <NInput
            v-model:value="password"
            type="password"
            placeholder="请输入密码"
            size="large"
            show-password-on="click"
            autocomplete="current-password"
            :disabled="loading"
            @keyup.enter="handleLogin"
          />
        </div>
        <NButton
          type="primary"
          block
          size="large"
          :loading="loading"
          attr-type="submit"
        >
          <template #icon>
            <NIcon :component="LogIn" />
          </template>
          登录
        </NButton>
        <div class="login-footer">
          <a class="login-link" @click="openChangePassword">修改密码</a>
        </div>
      </form>
    </NCard>

    <NModal v-model:show="showChangePasswordModal" :mask-closable="false">
      <NCard class="cp-card" :bordered="false" title="修改密码">
        <form class="login-form" @submit.prevent="handleCpSubmit">
          <div v-if="!cpOldPassword" class="login-field">
            <label class="login-label">旧密码</label>
            <NInput
              v-model:value="cpOldPassword"
              type="password"
              placeholder="请输入旧密码"
              size="large"
              show-password-on="click"
              autocomplete="current-password"
              :disabled="cpLoading"
            />
          </div>
          <div class="login-field">
            <label class="login-label">新密码</label>
            <NInput
              v-model:value="cpNewPassword"
              type="password"
              placeholder="请输入新密码（至少 6 位）"
              size="large"
              show-password-on="click"
              autocomplete="new-password"
              :disabled="cpLoading"
            />
          </div>
          <div class="login-field">
            <label class="login-label">确认新密码</label>
            <NInput
              v-model:value="cpConfirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              size="large"
              show-password-on="click"
              autocomplete="new-password"
              :disabled="cpLoading"
              @keyup.enter="handleCpSubmit"
            />
          </div>
          <NButton
            type="primary"
            block
            size="large"
            :loading="cpLoading"
            attr-type="submit"
          >
            <template #icon>
              <NIcon :component="KeyRound" />
            </template>
            确认修改
          </NButton>
        </form>
      </NCard>
    </NModal>
  </div>
</template>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f5f7fa;
}

[data-theme="dark"] .login-page {
  background: #1a1a2e;
}

.login-card {
  width: 380px;
  border-radius: 12px;
}

.login-header {
  text-align: center;
  margin-bottom: 24px;
}

.login-title {
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 4px;
  letter-spacing: -0.5px;
}

.login-subtitle {
  font-size: 14px;
  color: #999;
  margin: 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.login-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.login-label {
  font-size: 13px;
  font-weight: 500;
}

.login-footer {
  text-align: center;
  margin-top: -4px;
}

.login-link {
  font-size: 13px;
  color: #18a058;
  cursor: pointer;
}

.login-link:hover {
  text-decoration: underline;
}

.cp-card {
  width: 380px;
  border-radius: 12px;
}
</style>
