<script setup lang="ts">
import { computed } from 'vue'
import { useMessage } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'

const config = useConfigStore()
const server = useServerStore()
const message = useMessage()

const themeOptions = [
  { value: 'light', label: '浅色', icon: 'light-icon' },
  { value: 'dark', label: '深色', icon: 'dark-icon' },
  { value: 'auto', label: '跟随系统', icon: 'system-icon' }
] as const

const showTray = computed(() => config.draft?.ui.showTrayIcon ?? false)

async function save() {
  await config.save(server.status.running)
  message.success('设置已保存')
}
</script>

<template>
  <main class="settings-page">
    <header class="settings-page-header">
      <h1 class="settings-page-title">系统设置</h1>
      <p class="settings-page-subtitle">管理界面主题、启动行为、托盘显示与基础语言偏好</p>
    </header>

    <template v-if="config.draft">
      <section class="settings-section">
        <div class="settings-section-label">外观主题</div>
        <div class="settings-card">
          <div class="settings-row theme-row">
            <div class="row-info">
              <div class="row-label">颜色主题</div>
              <div class="row-desc">选择界面的显示风格，跟随系统会自动适配操作系统主题</div>
            </div>
            <div class="theme-selector" aria-label="主题切换">
              <button
                v-for="item in themeOptions"
                :key="item.value"
                class="theme-btn"
                :class="{ active: config.draft.ui.theme === item.value }"
                type="button"
                @click="config.draft.ui.theme = item.value"
              >
                <span class="theme-icon" :class="item.icon" />
                {{ item.label }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="settings-section">
        <div class="settings-section-label">启动行为</div>
        <div class="settings-card">
          <div class="settings-row">
            <div class="row-info">
              <div class="row-label">启动后自动最小化到托盘</div>
              <div class="row-desc">启动时不显示主窗口，直接缩小到系统托盘</div>
            </div>
            <label class="settings-toggle" title="启动后自动最小化到托盘">
              <input v-model="config.draft.ui.startMinimized" type="checkbox" />
              <span class="toggle-track" />
              <span class="toggle-thumb" />
            </label>
          </div>

          <div class="settings-row">
            <div class="row-info">
              <div class="row-label">启动后自动启动代理服务</div>
              <div class="row-desc">打开应用时立即启动代理服务，减少手动操作</div>
            </div>
            <label class="settings-toggle" title="启动后自动启动代理服务">
              <input v-model="config.draft.ui.autoStartProxy" type="checkbox" />
              <span class="toggle-track" />
              <span class="toggle-thumb" />
            </label>
          </div>
        </div>
      </section>

      <section class="settings-section">
        <div class="settings-section-label">托盘行为</div>
        <div class="settings-card">
          <div class="settings-row">
            <div class="row-info">
              <div class="row-label">显示系统托盘图标</div>
              <div class="row-desc">在系统托盘区域显示应用图标，便于查看运行状态</div>
            </div>
            <label class="settings-toggle" title="显示系统托盘图标">
              <input v-model="config.draft.ui.showTrayIcon" type="checkbox" />
              <span class="toggle-track" />
              <span class="toggle-thumb" />
            </label>
          </div>

          <div class="settings-row" :class="{ 'is-disabled': !showTray }">
            <div class="row-info">
              <div class="row-label">点击关闭按钮时最小化到托盘</div>
              <div class="row-desc">点击窗口关闭按钮时不退出程序，而是保持后台运行</div>
            </div>
            <label class="settings-toggle" title="点击关闭按钮时最小化到托盘">
              <input v-model="config.draft.ui.closeToTray" type="checkbox" :disabled="!showTray" />
              <span class="toggle-track" />
              <span class="toggle-thumb" />
            </label>
          </div>

          <div class="settings-row" :class="{ 'is-disabled': !showTray }">
            <div class="row-info">
              <div class="row-label">托盘菜单中显示当前服务状态和网卡 IP</div>
              <div class="row-desc">在托盘菜单顶部展示代理服务状态、监听端口和当前网卡 IP</div>
            </div>
            <label class="settings-toggle" title="托盘菜单中显示当前服务状态和网卡 IP">
              <input v-model="config.draft.ui.trayStatusAndIp" type="checkbox" :disabled="!showTray" />
              <span class="toggle-track" />
              <span class="toggle-thumb" />
            </label>
          </div>
        </div>
      </section>

      <section class="settings-section">
        <div class="settings-section-label">界面语言</div>
        <div class="settings-card">
          <div class="settings-row">
            <div class="row-info">
              <div class="row-label">显示语言</div>
              <div class="row-desc">当前版本暂时固定为简体中文</div>
            </div>
            <div class="select-wrap">
              <select v-model="config.draft.ui.language" class="select-input" disabled>
                <option value="zh-CN">zh-CN 简体中文</option>
              </select>
              <span class="select-arrow">⌄</span>
            </div>
          </div>
        </div>
      </section>

      <div class="settings-action-bar">
        <button class="settings-btn-save" type="button" :disabled="config.saving || !config.dirty" @click="save">
          保存设置
        </button>
        <button class="settings-btn-cancel" type="button" :disabled="config.saving || !config.dirty" @click="config.reset">
          取消
        </button>
        <span class="settings-save-hint">更改将在保存后生效</span>
      </div>
    </template>
  </main>
</template>
