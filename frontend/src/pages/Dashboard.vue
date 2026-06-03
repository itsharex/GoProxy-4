<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NAlert, NSpin } from 'naive-ui'
import { useConfigStore } from '../stores/config'
import { useLogStore } from '../stores/logs'
import { useServerStore } from '../stores/server'
import TrafficRateCanvas from '../components/TrafficRateCanvas.vue'
import { isWails } from '../backend/api'
import type { LogEntry } from '../types'

const server = useServerStore()
const config = useConfigStore()
const logs = useLogStore()

const lastBytes = ref({ up: 0, down: 0 })
const lastClientBytes = ref(new Map<string, { uploadBytes: number; downloadBytes: number }>())
const clientRates = ref(new Map<string, { uploadRate: number; downloadRate: number }>())
const chartTime = ref('--')
const logLevel = ref<'ALL' | LogEntry['level']>('ALL')
let timer: number | undefined

const protocolRows = computed(() => {
  const rows = new Map<string, { protocol: string; conns: number; upload: number; download: number }>()
  for (const conn of server.activeConnections) {
    const key = conn.protocol || 'UNKNOWN'
    const row = rows.get(key) ?? { protocol: key, conns: 0, upload: 0, download: 0 }
    row.conns += 1
    row.upload += conn.uploadBytes
    row.download += conn.downloadBytes
    rows.set(key, row)
  }
  return Array.from(rows.values()).sort((a, b) => b.conns - a.conns)
})

const logTabs: Array<{ label: string; value: 'ALL' | LogEntry['level'] }> = [
  { label: '全部', value: 'ALL' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'DEBUG', value: 'DEBUG' }
]

const maxConnections = computed(() => config.draft?.relay.maxConnections ?? 1000)
const uploadRate = computed(() => server.stats.uploadRate)
const downloadRate = computed(() => server.stats.downloadRate)
const totalTraffic = computed(() => server.stats.uploadBytes + server.stats.downloadBytes)
const clientRows = computed(() => {
  const grouped = new Map<
    string,
    {
      clientIp: string
      count: number
      uploadBytes: number
      downloadBytes: number
      uploadRate: number
      downloadRate: number
    }
  >()
  for (const conn of server.activeConnections) {
    const clientIp = normalizeClientIp(conn.clientAddr)
    const current = grouped.get(clientIp) ?? {
      clientIp,
      count: 0,
      uploadBytes: 0,
      downloadBytes: 0,
      uploadRate: 0,
      downloadRate: 0
    }
    current.count += 1
    current.uploadBytes += conn.uploadBytes
    current.downloadBytes += conn.downloadBytes
    grouped.set(clientIp, current)
  }
  for (const client of grouped.values()) {
    const rate = clientRates.value.get(client.clientIp)
    if (rate) {
      client.uploadRate = rate.uploadRate
      client.downloadRate = rate.downloadRate
    }
  }
  return [...grouped.values()].sort((a, b) => b.count - a.count || b.downloadRate - a.downloadRate).slice(0, 6)
})
const dashboardLogs = computed(() => {
  const entries = logs.entries
  if (!Array.isArray(entries)) return []
  return [...entries]
    .reverse()
    .filter((entry) => logLevel.value === 'ALL' || entry.level === logLevel.value)
    .slice(0, 40)
})

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

function connectionTraffic(upload: number, download: number): string {
  return `${formatBytes(upload)} ↑ ${formatBytes(download)} ↓`
}

function normalizeClientIp(address: string): string {
  if (!address) return '-'
  const bracketMatch = address.match(/^\[([^\]]+)\]:(\d+)$/)
  if (bracketMatch) return bracketMatch[1]
  const lastColon = address.lastIndexOf(':')
  if (lastColon > -1 && address.indexOf(':') === lastColon) {
    return address.slice(0, lastColon)
  }
  return address
}

function clientTrafficTitle(upload: number, download: number): string {
  return `总上行 ${formatBytes(upload)} / 总下行 ${formatBytes(download)}`
}

function shortTime(value: string): string {
  if (!value) return '--'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value.slice(11, 19) || value
  return parsed.toLocaleTimeString('zh-CN', { hour12: false })
}

function levelClass(level: LogEntry['level']) {
  return level.toLowerCase()
}

async function tick() {
  if (isWails()) await server.refresh()
  lastBytes.value = { up: server.stats.uploadBytes, down: server.stats.downloadBytes }
  updateClientRates()
  chartTime.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
}

function updateClientRates() {
  const totals = new Map<string, { uploadBytes: number; downloadBytes: number }>()
  for (const conn of server.activeConnections) {
    const clientIp = normalizeClientIp(conn.clientAddr)
    const current = totals.get(clientIp) ?? { uploadBytes: 0, downloadBytes: 0 }
    current.uploadBytes += conn.uploadBytes
    current.downloadBytes += conn.downloadBytes
    totals.set(clientIp, current)
  }

  const nextRates = new Map<string, { uploadRate: number; downloadRate: number }>()
  for (const [clientIp, total] of totals) {
    const previous = lastClientBytes.value.get(clientIp)
    nextRates.set(clientIp, {
      uploadRate: previous ? Math.max(0, total.uploadBytes - previous.uploadBytes) : 0,
      downloadRate: previous ? Math.max(0, total.downloadBytes - previous.downloadBytes) : 0
    })
  }
  clientRates.value = nextRates
  lastClientBytes.value = totals
}

onMounted(async () => {
  await Promise.all([server.refresh(), logs.load()])
  lastBytes.value = { up: server.stats.uploadBytes, down: server.stats.downloadBytes }
  updateClientRates()
  timer = window.setInterval(() => {
    void tick()
  }, 1000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <section class="dashboard">
    <NAlert v-if="server.error" type="error" class="page-alert">
      {{ server.error }}
    </NAlert>
    <NAlert v-if="!server.status.running" type="warning" class="page-alert">
      代理服务未启动，请点击右上角“启动服务”后再测试。
    </NAlert>

    <NSpin :show="server.loading">
      <div class="dashboard-stack">
        <div class="cards">
          <article class="card">
            <div class="card-label">当前连接数</div>
            <div class="card-val green">{{ server.status.activeConns }}</div>
            <div class="card-sub trend up">上限 {{ maxConnections }}</div>
          </article>
          <article class="card">
            <div class="card-label">上传速率</div>
            <div class="card-val blue rate-fixed">{{ padRate(uploadRate) }}</div>
            <div class="card-sub">本次运行 {{ formatBytes(server.stats.uploadBytes) }}</div>
          </article>
          <article class="card">
            <div class="card-label">下载速率</div>
            <div class="card-val amber rate-fixed">{{ padRate(downloadRate) }}</div>
            <div class="card-sub">本次运行 {{ formatBytes(server.stats.downloadBytes) }}</div>
          </article>
          <article class="card">
            <div class="card-label">总连接数</div>
            <div class="card-val">{{ server.status.totalConns }}</div>
            <div class="card-sub">总流量 {{ formatBytes(totalTraffic) }}</div>
          </article>
        </div>

        <div class="panels">
          <section class="panel">
            <div class="panel-head">
              <h3>实时流量速率</h3>
              <span class="tag ml">{{ chartTime }}</span>
            </div>
            <TrafficRateCanvas :data="server.trafficHistory" class="echarts-panel" />
          </section>
        </div>

        <section class="panel client-panel">
          <div class="panel-head">
            <h3>当前连接客户端</h3>
            <span class="tag ml">{{ clientRows.length }} CLIENTS</span>
          </div>
          <div class="client-table">
            <div class="client-table-head">
              <span>客户端 IP</span>
              <span>连接数</span>
              <span>实时上行</span>
              <span>实时下行</span>
              <span>总上 / 下行</span>
            </div>
            <div v-for="client in clientRows" :key="client.clientIp" class="client-item">
              <strong>{{ client.clientIp }}</strong>
              <span>{{ client.count }}</span>
              <span class="rate-fixed">{{ padRate(client.uploadRate) }}</span>
              <span class="rate-fixed">{{ padRate(client.downloadRate) }}</span>
              <span :title="clientTrafficTitle(client.uploadBytes, client.downloadBytes)">
                {{ connectionTraffic(client.uploadBytes, client.downloadBytes) }}
              </span>
            </div>
            <div v-if="clientRows.length === 0" class="empty-log compact">暂无客户端连接</div>
          </div>
        </section>

        <section class="panel protocol-panel">
          <div class="panel-head">
            <h3>协议分布</h3>
            <span class="tag">ACTIVE</span>
          </div>
          <table class="conn-table">
            <thead>
              <tr>
                <th>协议</th>
                <th>连接数</th>
                <th>上传</th>
                <th>下载</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="protocolRows.length === 0">
                <td colspan="4" class="table-empty">暂无活跃连接</td>
              </tr>
              <tr v-for="row in protocolRows" :key="row.protocol">
                <td>{{ row.protocol }}</td>
                <td>{{ row.conns }}</td>
                <td>{{ formatBytes(row.upload) }}</td>
                <td>{{ formatBytes(row.download) }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section class="panel dashboard-log-panel">
          <div class="tabs">
            <button
              v-for="tab in logTabs"
              :key="tab.value"
              class="tab"
              :class="{ active: logLevel === tab.value }"
              type="button"
              @click="logLevel = tab.value"
            >
              {{ tab.label }}
            </button>
          </div>
          <div class="panel-head log-head">
            <h3>实时日志</h3>
            <span class="tag ml">自动滚动</span>
          </div>
          <div class="dashboard-log-list">
            <div v-for="(entry, index) in dashboardLogs" :key="`${entry.time}-${index}`" class="log-item">
              <span class="log-time">{{ shortTime(entry.time) }}</span>
              <span class="log-lv" :class="levelClass(entry.level)">{{ entry.level }}</span>
              <span class="log-msg">{{ entry.message }}</span>
            </div>
            <div v-if="dashboardLogs.length === 0" class="empty-log compact">暂无日志</div>
          </div>
        </section>
      </div>
    </NSpin>
  </section>
</template>
