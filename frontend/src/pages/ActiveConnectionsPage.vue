<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NSpin } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useServerStore } from '../stores/server'
import { isWails } from '../backend/api'
import type { ActiveConnection } from '../types'

const server = useServerStore()
const config = useConfigStore()
let timer: number | undefined

const maxConnections = computed(() => config.draft?.relay.maxConnections ?? 1000)
const previousBytes = ref(new Map<number, { uploadBytes: number; downloadBytes: number }>())
const connectionRates = ref(new Map<number, { uploadRate: number; downloadRate: number }>())
const rows = computed(() =>
  server.activeConnections.map((conn) => ({
    ...conn,
    uploadRate: connectionRates.value.get(conn.id)?.uploadRate ?? 0,
    downloadRate: connectionRates.value.get(conn.id)?.downloadRate ?? 0
  }))
)

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let next = value / 1024
  let unit = units[0]
  for (let i = 1; i < units.length && next >= 1024; i += 1) {
    next /= 1024
    unit = units[i]
  }
  return `${next.toFixed(next >= 10 ? 1 : 2)} ${unit}`
}

function formatRate(value: number): string {
  return `${formatBytes(value)}/s`
}

function padRate(value: number): string {
  let num: string
  let unit: string
  if (value < 1024) {
    num = value.toFixed(1)
    unit = 'B'
  } else {
    const units = ['KB', 'MB', 'GB', 'TB']
    let next = value / 1024
    unit = units[0]
    for (let i = 1; i < units.length && next >= 1024; i += 1) {
      next /= 1024
      unit = units[i]
    }
    num = next.toFixed(next >= 10 ? 1 : 2)
  }
  return `${num.padStart(6)} ${unit}/s`.padEnd(13)
}

function formatProtocol(protocol: string): string {
  return protocol === 'socks5' ? 'SOCKS5' : 'HTTP'
}

function protocolClass(protocol: string): string {
  return protocol === 'socks5' ? 's5' : 'hc'
}

function shortTime(value: string): string {
  if (!value) return '--'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value.slice(11, 19) || value
  return parsed.toLocaleTimeString('zh-CN', { hour12: false })
}

function updateConnectionRates(connections: ActiveConnection[]) {
  const nextPrevious = new Map<number, { uploadBytes: number; downloadBytes: number }>()
  const nextRates = new Map<number, { uploadRate: number; downloadRate: number }>()

  for (const conn of connections) {
    const previous = previousBytes.value.get(conn.id)
    nextPrevious.set(conn.id, {
      uploadBytes: conn.uploadBytes,
      downloadBytes: conn.downloadBytes
    })
    nextRates.set(conn.id, {
      uploadRate: previous ? Math.max(0, conn.uploadBytes - previous.uploadBytes) : 0,
      downloadRate: previous ? Math.max(0, conn.downloadBytes - previous.downloadBytes) : 0
    })
  }

  previousBytes.value = nextPrevious
  connectionRates.value = nextRates
}

onMounted(async () => {
  await server.refresh()
  updateConnectionRates(server.activeConnections)
  timer = window.setInterval(async () => {
    if (isWails()) await server.refresh()
    updateConnectionRates(server.activeConnections)
  }, 1000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section class="connections-page">
    <NSpin :show="server.loading">
      <div class="panel active-panel">
        <div class="panel-head">
          <h3>活跃连接</h3>
          <span class="tag ml">{{ server.status.activeConns }} / {{ maxConnections }}</span>
        </div>
        <table class="conn-table active-conn-table">
          <thead>
            <tr>
              <th class="col-protocol">协议</th>
              <th class="col-client">客户端</th>
              <th class="col-target">目标</th>
              <th class="col-rule">命中规则</th>
              <th class="col-outbound">出口</th>
              <th class="col-upload-rate">实时上行</th>
              <th class="col-download-rate">实时下行</th>
              <th class="col-upload-total">累计上行</th>
              <th class="col-download-total">累计下行</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="conn in rows" :key="conn.id">
              <td class="col-protocol"><span class="proto" :class="protocolClass(conn.protocol)">{{ formatProtocol(conn.protocol) }}</span></td>
              <td class="col-client">{{ conn.clientAddr }}</td>
              <td class="col-target">{{ conn.targetAddr || '-' }}</td>
              <td class="col-rule">{{ conn.routeRuleName || '-' }}</td>
              <td class="col-outbound">{{ conn.outboundIface || conn.outboundIp || '-' }}</td>
              <td class="col-upload-rate rate-fixed">{{ padRate(conn.uploadRate) }}</td>
              <td class="col-download-rate rate-fixed">{{ padRate(conn.downloadRate) }}</td>
              <td class="col-upload-total">{{ formatBytes(conn.uploadBytes) }}</td>
              <td class="col-download-total">{{ formatBytes(conn.downloadBytes) }}</td>
            </tr>
            <tr v-if="rows.length === 0">
              <td colspan="9" class="table-empty">暂无活跃连接</td>
            </tr>
          </tbody>
        </table>
      </div>
    </NSpin>
  </section>
</template>

<style scoped>
.active-conn-table .col-protocol {
  width: 80px;
}

.active-conn-table .col-client {
  width: 150px;
}

.active-conn-table .col-target {
  width: auto;
}

.active-conn-table .col-rule {
  width: 100px;
}

.active-conn-table .col-outbound {
  width: 130px;
}

.active-conn-table .col-upload-rate {
  width: 100px;
}

.active-conn-table .col-download-rate {
  width: 100px;
}

.active-conn-table .col-upload-total {
  width: 100px;
}

.active-conn-table .col-download-total {
  width: 100px;
}
</style>
