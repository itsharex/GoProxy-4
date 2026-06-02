import type {
  ActiveConnection,
  AppConfig,
  LogEntry,
  NetworkInterface,
  RouteFileInfo,
  RouteRuleSet,
  SSESnapshot,
  ServerStatus,
  StatsSnapshot
} from '../types'

export type EventDisposer = () => void

type ApiResponse<T> = {
  ok: boolean
  data: T
  error?: string
}

const API_BASE = '/api/v1'
const WS_BASE = getWsBase()

let wsInstance: WebSocket | null = null
let wsReconnectTimer: ReturnType<typeof setTimeout> | null = null
const eventListeners = new Map<string, Set<(data: any) => void>>()

function getWsBase(): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws`
}

function getToken(): string | null {
  return localStorage.getItem('goproxy_token')
}

function setToken(token: string) {
  localStorage.setItem('goproxy_token', token)
}

function clearToken() {
  localStorage.removeItem('goproxy_token')
}

async function request<T>(method: string, path: string, body?: any): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined
  })
  const json: ApiResponse<T> = await res.json()
  if (res.status === 401) {
    clearToken()
    const loginPath = '/#/login'
    if (location.hash !== loginPath) {
      location.hash = loginPath
    }
    throw new Error(json.error || '认证已过期，请重新登录')
  }
  if (!json.ok) {
    throw new Error(json.error || '请求失败')
  }
  return json.data
}

function connectWebSocket() {
  if (wsInstance && wsInstance.readyState === WebSocket.OPEN) return
  const token = getToken()
  if (!token) return

  const url = `${WS_BASE}?token=${encodeURIComponent(token)}`
  const ws = new WebSocket(url)

  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data)
      if (msg.event === 'ping') {
        ws.send(JSON.stringify({ event: 'pong' }))
        return
      }
      const listeners = eventListeners.get(msg.event)
      if (listeners) {
        listeners.forEach((cb) => cb(msg.data))
      }
    } catch { /* ignore parse errors */ }
  }

  ws.onclose = () => {
    wsInstance = null
    scheduleReconnect()
  }

  ws.onerror = () => {
    ws.close()
  }

  wsInstance = ws
}

function scheduleReconnect() {
  if (wsReconnectTimer) return
  wsReconnectTimer = setTimeout(() => {
    wsReconnectTimer = null
    if (getToken()) {
      connectWebSocket()
    }
  }, 3000)
}

export function isWails(): boolean {
  return false
}

export function isWebLoggedIn(): boolean {
  return !!getToken()
}

export async function webLogin(username: string, password: string): Promise<{ token: string; expiresAt: string }> {
  clearToken()
  const res = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  })
  const json: ApiResponse<{ token: string; expiresAt: string }> = await res.json()
  if (!json.ok) {
    throw new Error(json.error || '登录失败')
  }
  setToken(json.data.token)
  connectWebSocket()
  return json.data
}

export async function webCheckAuth(): Promise<boolean> {
  try {
    await request<boolean>('GET', '/auth/check')
    return true
  } catch {
    return false
  }
}

export async function getConfig() {
  return request<AppConfig>('GET', '/config')
}

export async function getLocalIPAddresses() {
  return request<string[]>('GET', '/platform/ips')
}

export async function getNetworkInterfaces() {
  return request<NetworkInterface[]>('GET', '/platform/interfaces')
}

export async function saveConfig(config: AppConfig) {
  return request<void>('PUT', '/config', config)
}

export async function listRouteFiles() {
  return request<RouteFileInfo[]>('GET', '/routes/files')
}

export async function loadRouteFile(name: string) {
  return request<RouteRuleSet>('GET', `/routes/files/${encodeURIComponent(name)}`)
}

export async function saveRouteFile(name: string, ruleSet: RouteRuleSet) {
  return request<void>('PUT', `/routes/files/${encodeURIComponent(name)}`, ruleSet)
}

export async function createRouteFile(name: string) {
  return request<void>('POST', '/routes/files', { name })
}

export async function deleteRouteFile(name: string) {
  return request<void>('DELETE', `/routes/files/${encodeURIComponent(name)}`)
}

export async function setActiveRouteFile(name: string) {
  return request<void>('PUT', '/routes/active', { name })
}

export async function startServer() {
  return request<ServerStatus>('POST', '/server/start')
}

export async function stopServer() {
  return request<ServerStatus>('POST', '/server/stop')
}

export async function getServerStatus() {
  return request<ServerStatus>('GET', '/server/status')
}

export async function getStats() {
  return request<StatsSnapshot>('GET', '/server/stats')
}

export async function getActiveConnections() {
  return request<ActiveConnection[]>('GET', '/server/connections')
}

export async function getRecentLogs(n: number) {
  return request<LogEntry[]>('GET', `/logs?n=${n}`)
}

export async function clearLogs() {
  return request<void>('POST', '/logs/clear')
}

export async function setAuthEnabled(enabled: boolean) {
  return request<void>('PUT', '/auth/enabled', { enabled })
}

export async function addUser(username: string, password: string) {
  return request<void>('POST', '/auth/users', { username, password })
}

export async function removeUser(username: string) {
  return request<void>('DELETE', `/auth/users/${encodeURIComponent(username)}`)
}

export async function resetUserPassword(username: string, password: string) {
  return request<void>('PUT', `/auth/users/${encodeURIComponent(username)}/password`, { password })
}

export function getTrayState() {
  return Promise.resolve({ enabled: false, visible: false, platform: 'web', supportsMenu: false, hideDescription: '' })
}

export function showWindow() {
  return Promise.resolve()
}

export function hideToTray() {
  return Promise.resolve()
}

export function quitApp() {
  clearToken()
  location.reload()
}

export async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
}

export function onEvent<T>(eventName: string, callback: (payload: T) => void): EventDisposer {
  if (!eventListeners.has(eventName)) {
    eventListeners.set(eventName, new Set())
  }
  const listeners = eventListeners.get(eventName)!
  listeners.add(callback as (data: any) => void)

  if (!wsInstance && getToken()) {
    connectWebSocket()
  }

  return () => {
    listeners.delete(callback as (data: any) => void)
  }
}

export function initWebWebSocket() {
  if (getToken()) {
    connectWebSocket()
  }
}

let esInstance: EventSource | null = null
let esReconnectTimer: ReturnType<typeof setTimeout> | null = null
const snapshotListeners = new Set<(snapshot: SSESnapshot) => void>()

function connectSSE() {
  if (esInstance && esInstance.readyState === EventSource.OPEN) return
  const token = getToken()
  if (!token) return

  esInstance?.close()
  const url = `${API_BASE}/events?token=${encodeURIComponent(token)}`
  esInstance = new EventSource(url)

  esInstance.addEventListener('snapshot', (e: MessageEvent) => {
    try {
      const snapshot: SSESnapshot = JSON.parse(e.data)
      snapshotListeners.forEach((cb) => cb(snapshot))
    } catch { /* ignore parse errors */ }
  })

  esInstance.onerror = () => {
    esInstance?.close()
    esInstance = null
    if (!esReconnectTimer) {
      esReconnectTimer = setTimeout(() => {
        esReconnectTimer = null
        if (getToken()) connectSSE()
      }, 3000)
    }
  }
}

export function onServerSnapshot(callback: (snapshot: SSESnapshot) => void): EventDisposer {
  snapshotListeners.add(callback)
  if (!esInstance && getToken()) {
    connectSSE()
  }
  return () => {
    snapshotListeners.delete(callback)
  }
}
