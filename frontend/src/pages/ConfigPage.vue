<script setup lang="ts">
import {
  ArrowRightLeft,
  CircleHelp,
  Clock3,
  FileText,
  Globe,
  HardDrive,
  Save,
  Server,
  Shield,
  RotateCcw,
  TimerReset,
  Users,
  Wifi,
  Zap
} from 'lucide-vue-next'
import {
  NAlert,
  NButton,
  NIcon,
  NInput,
  NInputNumber,
  NSelect,
  NSpin,
  NSwitch,
  useMessage
} from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'

const config = useConfigStore()
const server = useServerStore()
const message = useMessage()

const logLevels = [
  { label: 'debug', value: 'debug' },
  { label: 'info', value: 'info' },
  { label: 'warn', value: 'warn' },
  { label: 'error', value: 'error' }
]

const logOutputs = [
  { label: '文件 + 控制台', value: 'both' },
  { label: '仅文件', value: 'file' },
  { label: '仅控制台', value: 'console' }
]

async function save() {
  await config.save(server.status.running)
  await server.refresh()
  message.success('配置已保存')
}
</script>

<template>
  <section class="service-config-page">
    <div class="page-shell">
      <div class="page-header">
        <div class="page-header-main">
          <div class="page-header-icon">
            <NIcon :component="Server" />
          </div>
          <div>
            <h2 class="page-title">服务配置</h2>
            <p class="page-subtitle">配置代理服务的入站协议和转发参数</p>
          </div>
        </div>
        <div class="page-header-actions">
          <NButton secondary :disabled="!config.dirty" @click="config.reset">
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

      <NAlert v-if="config.error" type="error" class="page-alert">
        {{ config.error }}
      </NAlert>
      <NAlert v-if="config.restartRequired" type="warning" class="page-alert">
        监听配置已保存，重启服务后生效。
      </NAlert>

      <NSpin :show="config.loading">
        <template v-if="config.draft">
          <div class="config-stack">
            <section class="config-card">
              <div class="card-head">
                <div class="card-title-wrap">
                  <div class="card-icon listener">
                    <NIcon :component="Globe" />
                  </div>
                  <div>
                    <div class="card-title">入站协议</div>
                    <div class="card-subtitle">LISTENER · 可同时开启多种协议监听</div>
                  </div>
                </div>
              </div>

              <div class="card-content protocol-stack">
                <div
                  class="protocol-block socks"
                  :class="{ disabled: !config.draft.server.socks5.enabled }"
                >
                  <div class="protocol-head">
                    <div class="protocol-meta">
                      <div class="protocol-icon socks">
                        <NIcon :component="Shield" />
                      </div>
                      <div>
                        <div class="protocol-title-row">
                          <span class="protocol-title">SOCKS5</span>
                          <span v-if="config.draft.server.socks5.enabled" class="protocol-badge socks">运行中</span>
                        </div>
                        <span class="protocol-note">支持 TCP 转发与 DNS 解析</span>
                      </div>
                    </div>
                    <NSwitch v-model:value="config.draft.server.socks5.enabled" />
                  </div>

                  <div v-show="config.draft.server.socks5.enabled" class="protocol-fields socks">
                    <div class="field-row">
                      <div class="field-label">
                        <span>监听地址</span>
                        <span class="tip-icon" title="SOCKS5 服务监听的 IP 地址，0.0.0.0 表示监听所有网卡">
                          <NIcon :component="CircleHelp" />
                        </span>
                      </div>
                      <div class="field-control">
                        <div class="input-line">
                          <NIcon :component="Globe" class="field-leading-icon" />
                          <NInput v-model:value="config.draft.server.socks5.host" class="mono-input" />
                        </div>
                      </div>
                    </div>

                    <div class="field-row">
                      <div class="field-label">
                        <span>端口</span>
                      </div>
                      <div class="field-control">
                        <div class="input-line compact">
                          <NIcon :component="Zap" class="field-leading-icon" />
                          <NInputNumber
                            v-model:value="config.draft.server.socks5.port"
                            :min="1"
                            :max="65535"
                            class="port-input"
                          />
                          <span class="field-suffix">TCP</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div
                  class="protocol-block http"
                  :class="{ disabled: !config.draft.server.http.enabled }"
                >
                  <div class="protocol-head">
                    <div class="protocol-meta">
                      <div class="protocol-icon http">
                        <NIcon :component="Wifi" />
                      </div>
                      <div>
                        <div class="protocol-title-row">
                          <span class="protocol-title">HTTP CONNECT</span>
                          <span v-if="config.draft.server.http.enabled" class="protocol-badge http">运行中</span>
                        </div>
                        <span class="protocol-note">HTTP / HTTPS 代理模式</span>
                      </div>
                    </div>
                    <NSwitch v-model:value="config.draft.server.http.enabled" />
                  </div>

                  <div v-show="config.draft.server.http.enabled" class="protocol-fields http">
                    <div class="field-row">
                      <div class="field-label">
                        <span>监听地址</span>
                        <span class="tip-icon" title="HTTP 代理服务监听的 IP 地址，0.0.0.0 表示监听所有网卡">
                          <NIcon :component="CircleHelp" />
                        </span>
                      </div>
                      <div class="field-control">
                        <div class="input-line">
                          <NIcon :component="Globe" class="field-leading-icon" />
                          <NInput v-model:value="config.draft.server.http.host" class="mono-input" />
                        </div>
                      </div>
                    </div>

                    <div class="field-row">
                      <div class="field-label">
                        <span>端口</span>
                      </div>
                      <div class="field-control">
                        <div class="input-line compact">
                          <NIcon :component="Zap" class="field-leading-icon" />
                          <NInputNumber
                            v-model:value="config.draft.server.http.port"
                            :min="1"
                            :max="65535"
                            class="port-input"
                          />
                          <span class="field-suffix">TCP</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <section class="config-card config-card-spaced">
              <div class="card-head">
                <div class="card-title-wrap">
                  <div class="card-icon relay">
                    <NIcon :component="ArrowRightLeft" />
                  </div>
                  <div>
                    <div class="card-title">转发参数</div>
                    <div class="card-subtitle">RELAY</div>
                  </div>
                </div>
              </div>

              <div class="card-content">
                <div class="relay-surface">
                  <div class="field-row">
                    <div class="field-label">
                      <span>目标建连超时</span>
                      <span class="tip-icon" title="与目标服务器建立 TCP 连接的最大等待时间">
                        <NIcon :component="CircleHelp" />
                      </span>
                    </div>
                    <div class="field-control">
                      <div class="input-line compact">
                        <NIcon :component="TimerReset" class="field-leading-icon" />
                        <NInputNumber v-model:value="config.draft.relay.dialTimeoutSec" :min="1" class="short-input" />
                        <span class="field-suffix">秒</span>
                      </div>
                    </div>
                  </div>

                  <div class="field-row">
                    <div class="field-label">
                      <span>握手 / 读写超时</span>
                      <span class="tip-icon" title="握手和读写操作的最大等待时间">
                        <NIcon :component="CircleHelp" />
                      </span>
                    </div>
                    <div class="field-control">
                      <div class="input-line compact">
                        <NIcon :component="ArrowRightLeft" class="field-leading-icon" />
                        <NInputNumber v-model:value="config.draft.relay.readTimeoutSec" :min="1" class="short-input" />
                        <span class="field-suffix">秒</span>
                      </div>
                    </div>
                  </div>

                  <div class="field-row">
                    <div class="field-label">
                      <span>最大并发连接数</span>
                      <span class="tip-icon" title="允许同时建立的最大连接数量">
                        <NIcon :component="CircleHelp" />
                      </span>
                    </div>
                    <div class="field-control">
                      <div class="input-line compact">
                        <NIcon :component="Users" class="field-leading-icon" />
                        <NInputNumber v-model:value="config.draft.relay.maxConnections" :min="1" class="short-input" />
                        <span class="field-suffix">个连接</span>
                      </div>
                    </div>
                  </div>

                  <div class="field-row">
                    <div class="field-label">
                      <span>Keep-Alive 间隔</span>
                      <span class="tip-icon" title="TCP Keep-Alive 探测包的发送间隔">
                        <NIcon :component="CircleHelp" />
                      </span>
                    </div>
                    <div class="field-control">
                      <div class="input-line compact">
                        <NIcon :component="Clock3" class="field-leading-icon" />
                        <NInputNumber v-model:value="config.draft.relay.keepaliveSec" :min="1" class="short-input" />
                        <span class="field-suffix">秒</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <section class="config-card config-card-spaced">
              <div class="card-head">
                <div class="card-title-wrap">
                  <div class="card-icon logging">
                    <NIcon :component="FileText" />
                  </div>
                  <div>
                    <div class="card-title">日志参数</div>
                    <div class="card-subtitle">LOGGING</div>
                  </div>
                </div>
              </div>

              <div class="card-content">
                <div class="relay-surface">
                  <div class="field-row">
                    <div class="field-label">
                      <span>级别</span>
                    </div>
                    <div class="field-control">
                      <NSelect v-model:value="config.draft.log.level" :options="logLevels" />
                    </div>
                  </div>

                  <div class="field-row">
                    <div class="field-label">
                      <span>输出</span>
                    </div>
                    <div class="field-control">
                      <NSelect v-model:value="config.draft.log.output" :options="logOutputs" />
                    </div>
                  </div>

                  <div class="field-row">
                    <div class="field-label">
                      <span>单文件大小</span>
                    </div>
                    <div class="field-control">
                      <div class="input-line compact">
                        <NIcon :component="HardDrive" class="field-leading-icon" />
                        <NInputNumber v-model:value="config.draft.log.maxSizeMb" :min="1" class="short-input" />
                        <span class="field-suffix">MB</span>
                      </div>
                    </div>
                  </div>

                  <div class="field-row">
                    <div class="field-label">
                      <span>备份数量</span>
                    </div>
                    <div class="field-control">
                      <div class="input-line compact">
                        <NIcon :component="FileText" class="field-leading-icon" />
                        <NInputNumber v-model:value="config.draft.log.maxBackups" :min="0" class="short-input" />
                        <span class="field-suffix">份</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <div class="action-bar action-bar-spaced">
              <p class="action-hint">修改配置后请保存，以使更改生效</p>
              <div class="action-buttons">
                <NButton secondary :disabled="!config.dirty" @click="config.reset">
                  <template #icon>
                    <NIcon :component="TimerReset" />
                  </template>
                  重置
                </NButton>
                <NButton type="primary" :loading="config.saving" :disabled="!config.dirty" @click="save">
                  <template #icon>
                    <NIcon :component="Save" />
                  </template>
                  保存配置
                </NButton>
              </div>
            </div>
          </div>
        </template>
      </NSpin>
    </div>
  </section>
</template>

<style scoped>
.service-config-page {
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

.card-icon.listener {
  color: #059669;
  background: rgba(16, 185, 129, 0.12);
}

.card-icon.relay {
  color: #d97706;
  background: rgba(245, 158, 11, 0.12);
}

.card-icon.logging {
  color: #52525b;
  background: rgba(113, 113, 122, 0.12);
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

.protocol-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.protocol-block {
  border-radius: 12px;
  border: 2px solid;
  transition: border-color 0.2s ease, background-color 0.2s ease, opacity 0.2s ease;
}

.protocol-block.socks {
  border-color: rgba(16, 185, 129, 0.26);
  background: rgba(16, 185, 129, 0.07);
}

.protocol-block.http {
  border-color: rgba(14, 165, 233, 0.26);
  background: rgba(14, 165, 233, 0.07);
}

.protocol-block.disabled {
  border-color: var(--border);
  background: color-mix(in srgb, var(--panel) 88%, var(--fg-soft) 12%);
}

.protocol-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
}

.protocol-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.protocol-icon {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  flex: 0 0 auto;
}

.protocol-icon.socks {
  color: #059669;
  background: rgba(16, 185, 129, 0.14);
}

.protocol-icon.http {
  color: #0284c7;
  background: rgba(14, 165, 233, 0.14);
}

.protocol-block.disabled .protocol-icon {
  color: var(--fg-soft);
  background: color-mix(in srgb, var(--panel) 78%, var(--fg-soft) 22%);
}

.protocol-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.protocol-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--fg);
}

.protocol-block.disabled .protocol-title,
.protocol-block.disabled .protocol-note {
  color: var(--fg-soft);
}

.protocol-note {
  display: inline-block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--fg-soft);
}

.protocol-badge {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
}

.protocol-badge.socks {
  color: #047857;
  background: rgba(16, 185, 129, 0.16);
}

.protocol-badge.http {
  color: #0369a1;
  background: rgba(14, 165, 233, 0.16);
}

.protocol-fields {
  padding: 14px 16px 16px;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.protocol-fields.socks {
  border-top-color: rgba(16, 185, 129, 0.22);
}

.protocol-fields.http {
  border-top-color: rgba(14, 165, 233, 0.22);
}

.relay-surface {
  border-radius: 10px;
  border: 1px solid var(--border);
  background: color-mix(in srgb, var(--panel) 90%, var(--fg-soft) 10%);
  padding: 18px 16px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.field-row {
  display: grid;
  grid-template-columns: minmax(140px, 180px) minmax(0, 1fr);
  align-items: center;
  gap: 16px;
}

.field-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 14px;
  color: var(--fg-soft);
}

.tip-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--fg-soft);
  font-size: 14px;
}

.field-control {
  min-width: 0;
}

.input-line {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.input-line.compact {
  gap: 8px;
}

.field-leading-icon {
  flex: 0 0 auto;
  color: var(--fg-soft);
  font-size: 16px;
}

.field-suffix {
  font-size: 12px;
  color: var(--fg-soft);
  white-space: nowrap;
}

.mono-input :deep(.n-input__input-el),
.port-input :deep(input),
.short-input :deep(input) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.port-input,
.short-input {
  width: 132px;
  flex: 0 0 auto;
}

.mono-input {
  width: 100%;
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
  margin-top: 24px;
}

.action-hint {
  margin: 0;
  font-size: 12px;
  color: var(--fg-soft);
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 12px;
}

.service-config-page :deep(.n-input),
.service-config-page :deep(.n-base-selection),
.service-config-page :deep(.n-input-number) {
  --n-border-radius: 8px !important;
}

.service-config-page :deep(.n-input .n-input__border),
.service-config-page :deep(.n-base-selection .n-base-selection__border),
.service-config-page :deep(.n-input-number .n-input-wrapper) {
  border-color: var(--border) !important;
}

.service-config-page :deep(.n-input:hover .n-input__border),
.service-config-page :deep(.n-base-selection:hover .n-base-selection__border),
.service-config-page :deep(.n-input-number:hover .n-input-wrapper) {
  border-color: color-mix(in srgb, var(--fg) 22%, var(--border) 78%) !important;
}

@media (max-width: 900px) {
  .page-shell {
    max-width: 100%;
  }
}

@media (max-width: 720px) {
  .page-header,
  .page-header-main,
  .card-head,
  .action-bar {
    flex-direction: column;
    align-items: flex-start;
  }

  .page-header-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .field-row {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .field-control,
  .mono-input {
    width: 100%;
  }

  .action-buttons {
    width: 100%;
    justify-content: flex-end;
    flex-wrap: wrap;
  }
}
</style>
