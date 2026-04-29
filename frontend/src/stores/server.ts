import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getActiveConnections, getServerStatus, getStats, startServer, stopServer } from '../backend/api'
import type { ActiveConnection, ServerStatus, StatsSnapshot, TrafficSample } from '../types'
import { friendlyError } from '../utils/errors'

const emptyStatus: ServerStatus = {
  running: false,
  startedAt: '',
  socks5Addr: '',
  httpAddr: '',
  activeConns: 0,
  totalConns: 0
}

const emptyStats: StatsSnapshot = {
  activeConns: 0,
  totalConns: 0,
  uploadBytes: 0,
  downloadBytes: 0,
  uploadRate: 0,
  downloadRate: 0,
  authFailures: 0
}

export const useServerStore = defineStore('server', () => {
  const status = ref<ServerStatus>({ ...emptyStatus })
  const stats = ref<StatsSnapshot>({ ...emptyStats })
  const trafficHistory = ref<TrafficSample[]>([])
  const activeConnections = ref<ActiveConnection[]>([])
  const loading = ref(false)
  const error = ref('')

  const totalBytes = computed(() => stats.value.uploadBytes + stats.value.downloadBytes)

  async function refresh() {
    error.value = ''
    try {
      status.value = await getServerStatus()
      setStats(await getStats())
      activeConnections.value = await getActiveConnections()
    } catch (err) {
      error.value = friendlyError(err)
    }
  }

  async function start() {
    loading.value = true
    error.value = ''
    try {
      await startServer()
      await refresh()
    } catch (err) {
      error.value = friendlyError(err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function stop() {
    loading.value = true
    error.value = ''
    try {
      await stopServer()
      await refresh()
    } catch (err) {
      error.value = friendlyError(err)
      throw err
    } finally {
      loading.value = false
    }
  }

  function setStatus(next: ServerStatus) {
    status.value = next
  }

  function setStats(next: StatsSnapshot) {
    stats.value = next
    const sample: TrafficSample = {
      ...next,
      time: new Date().toLocaleTimeString('zh-CN', { hour12: false })
    }
    trafficHistory.value = [...trafficHistory.value, sample].slice(-60)
  }

  return {
    status,
    stats,
    trafficHistory,
    activeConnections,
    loading,
    error,
    totalBytes,
    refresh,
    start,
    stop,
    setStatus,
    setStats
  }
})
