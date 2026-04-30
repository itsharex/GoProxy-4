<script setup lang="ts">
import { computed } from 'vue'
import {
  MonitorCog,
  Moon,
  RotateCcw,
  Save,
  ServerCog,
  Settings,
  Sun,
  Languages,
  PanelTop,
  Power
} from 'lucide-vue-next'
import { NButton, NIcon, NSelect, NSwitch, useMessage } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'

const config = useConfigStore()
const server = useServerStore()
const message = useMessage()

const themeOptions = [
  { value: 'light', label: '浅色', icon: Sun },
  { value: 'dark', label: '深色', icon: Moon },
  { value: 'auto', label: '跟随系统', icon: MonitorCog }
] as const

const showTray = computed(() => config.draft?.ui.showTrayIcon ?? false)

async function save() {
  await config.save(server.status.running)
  message.success('设置已保存')
}
</script>

<template>
  <section class="settings-unified-page">
    <div class="page-shell" v-if="config.draft">
      <div class="page-header">
        <div class="page-header-main">
          <div class="page-header-icon">
            <NIcon :component="Settings" />
          </div>
          <div>
            <h2 class="page-title">应用设置</h2>
            <p class="page-subtitle">管理界面主题、启动行为、托盘显示与基础语言偏好</p>
          </div>
        </div>
        <div class="page-header-actions">
          <NButton secondary :disabled="config.saving || !config.dirty" @click="config.reset">
            <template #icon>
              <NIcon :component="RotateCcw" />
            </template>
            重置
          </NButton>
          <NButton type="primary" :loading="config.saving" :disabled="!config.dirty" @click="save">
            <template #icon>
              <NIcon :component="Save" />
            </template>
            保存
          </NButton>
        </div>
      </div>

      <section class="config-card">
        <div class="card-head">
          <div class="card-title-wrap">
            <div class="card-icon theme">
              <NIcon :component="MonitorCog" />
            </div>
            <div>
              <div class="card-title">外观主题</div>
              <div class="card-subtitle">THEME</div>
            </div>
          </div>
        </div>

        <div class="card-content">
          <div class="surface-block">
            <div class="field-row theme-layout">
              <div class="field-label-group">
                <div class="field-title">颜色主题</div>
                <div class="field-desc">选择界面的显示风格，跟随系统会自动适配操作系统主题</div>
              </div>
              <div class="theme-selector">
                <button
                  v-for="item in themeOptions"
                  :key="item.value"
                  class="theme-button"
                  :class="{ active: config.draft.ui.theme === item.value }"
                  type="button"
                  @click="config.draft.ui.theme = item.value"
                >
                  <NIcon :component="item.icon" />
                  <span>{{ item.label }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="config-card config-card-spaced">
        <div class="card-head">
          <div class="card-title-wrap">
            <div class="card-icon startup">
              <NIcon :component="Power" />
            </div>
            <div>
              <div class="card-title">启动行为</div>
              <div class="card-subtitle">STARTUP</div>
            </div>
          </div>
        </div>

        <div class="card-content">
          <div class="surface-block settings-list">
            <div class="field-row field-row-switch">
              <div class="field-label-group">
                <div class="field-title">启动后自动最小化到托盘</div>
                <div class="field-desc">启动时不显示主窗口，直接缩小到系统托盘</div>
              </div>
              <NSwitch v-model:value="config.draft.ui.startMinimized" />
            </div>

            <div class="field-row field-row-switch">
              <div class="field-label-group">
                <div class="field-title">启动后自动启动代理服务</div>
                <div class="field-desc">打开应用时立刻启动代理服务，减少手动操作</div>
              </div>
              <NSwitch v-model:value="config.draft.ui.autoStartProxy" />
            </div>
          </div>
        </div>
      </section>

      <section class="config-card config-card-spaced">
        <div class="card-head">
          <div class="card-title-wrap">
            <div class="card-icon tray">
              <NIcon :component="PanelTop" />
            </div>
            <div>
              <div class="card-title">托盘行为</div>
              <div class="card-subtitle">TRAY</div>
            </div>
          </div>
        </div>

        <div class="card-content">
          <div class="surface-block settings-list">
            <div class="field-row field-row-switch">
              <div class="field-label-group">
                <div class="field-title">显示系统托盘图标</div>
                <div class="field-desc">在系统托盘区域显示应用图标，便于查看运行状态</div>
              </div>
              <NSwitch v-model:value="config.draft.ui.showTrayIcon" />
            </div>

            <div class="field-row field-row-switch" :class="{ disabled: !showTray }">
              <div class="field-label-group">
                <div class="field-title">点击关闭按钮时最小化到托盘</div>
                <div class="field-desc">点击窗口关闭按钮时不退出程序，而是保持后台运行</div>
              </div>
              <NSwitch v-model:value="config.draft.ui.closeToTray" :disabled="!showTray" />
            </div>

            <div class="field-row field-row-switch" :class="{ disabled: !showTray }">
              <div class="field-label-group">
                <div class="field-title">托盘菜单中显示当前服务状态和网卡 IP</div>
                <div class="field-desc">在托盘菜单顶部展示代理服务状态、监听端口和当前网卡 IP</div>
              </div>
              <NSwitch v-model:value="config.draft.ui.trayStatusAndIp" :disabled="!showTray" />
            </div>
          </div>
        </div>
      </section>

      <section class="config-card config-card-spaced">
        <div class="card-head">
          <div class="card-title-wrap">
            <div class="card-icon language">
              <NIcon :component="Languages" />
            </div>
            <div>
              <div class="card-title">界面语言</div>
              <div class="card-subtitle">LANGUAGE</div>
            </div>
          </div>
        </div>

        <div class="card-content">
          <div class="surface-block">
            <div class="field-row">
              <div class="field-label-group">
                <div class="field-title">显示语言</div>
                <div class="field-desc">当前版本暂时固定为简体中文</div>
              </div>
              <div class="field-control">
                <NSelect
                  v-model:value="config.draft.ui.language"
                  :options="[{ label: 'zh-CN 简体中文', value: 'zh-CN' }]"
                  disabled
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      <div class="action-bar action-bar-spaced">
        <p class="action-hint">修改设置后请保存，以使更改生效</p>
        <div class="page-header-actions">
          <NButton secondary :disabled="config.saving || !config.dirty" @click="config.reset">
            <template #icon>
              <NIcon :component="RotateCcw" />
            </template>
            重置
          </NButton>
          <NButton type="primary" :loading="config.saving" :disabled="!config.dirty" @click="save">
            <template #icon>
              <NIcon :component="Save" />
            </template>
            保存设置
          </NButton>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.settings-unified-page {
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

.config-card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}

.config-card-spaced {
  margin-top: 06px;
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

.card-icon.theme {
  color: #7c3aed;
  background: rgba(124, 58, 237, 0.12);
}

.card-icon.startup {
  color: #d97706;
  background: rgba(245, 158, 11, 0.12);
}

.card-icon.tray {
  color: #0284c7;
  background: rgba(14, 165, 233, 0.12);
}

.card-icon.language {
  color: #059669;
  background: rgba(16, 185, 129, 0.12);
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

.surface-block {
  border-radius: 10px;
  border: 1px solid var(--border);
  background: color-mix(in srgb, var(--panel) 90%, var(--fg-soft) 10%);
  padding: 18px 16px;
}

.settings-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.field-row {
  display: grid;
  grid-template-columns: minmax(180px, 240px) minmax(0, 1fr);
  align-items: center;
  gap: 16px;
}

.field-row-switch {
  grid-template-columns: minmax(0, 1fr) auto;
}

.field-label-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
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

.field-control {
  min-width: 0;
}

.theme-layout {
  grid-template-columns: minmax(180px, 240px) minmax(0, 1fr);
}

.theme-selector {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.theme-button {
  border: 1px solid var(--border);
  background: var(--panel);
  color: var(--fg-soft);
  border-radius: 10px;
  height: 38px;
  padding: 0 14px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.theme-button.active {
  color: var(--fg);
  border-color: color-mix(in srgb, var(--fg) 22%, var(--border) 78%);
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.08);
}

.disabled {
  opacity: 0.58;
}

.action-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--panel);
  padding: 16px 18px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.04);
}

.action-bar-spaced {
  margin-top: 18px;
}

.action-hint {
  margin: 0;
  font-size: 12px;
  color: var(--fg-soft);
}

.settings-unified-page :deep(.n-base-selection) {
  --n-border-radius: 8px !important;
}

@media (max-width: 720px) {
  .page-header,
  .page-header-main,
  .action-bar,
  .field-row,
  .field-row-switch,
  .theme-layout {
    grid-template-columns: 1fr;
    flex-direction: column;
    align-items: flex-start;
  }

  .page-header-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
