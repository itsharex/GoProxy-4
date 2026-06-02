<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NCard, NInput, NIcon, useMessage } from 'naive-ui'
import { LogIn } from 'lucide-vue-next'
import { webLogin } from '../backend/api'

const username = ref('')
const password = ref('')
const loading = ref(false)
const message = useMessage()

async function handleLogin() {
  if (!username.value.trim() || !password.value) {
    message.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    await webLogin(username.value.trim(), password.value)
    message.success('登录成功')
    location.hash = ''
    setTimeout(() => location.reload(), 300)
  } catch (err: any) {
    message.error(err?.message || '登录失败')
  } finally {
    loading.value = false
  }
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
      </form>
    </NCard>
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
</style>
