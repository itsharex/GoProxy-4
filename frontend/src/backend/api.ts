import type {
  ActiveConnection,
  AppConfig,
  ChangePasswordResponse,
  CheckAuthResponse,
  LogEntry,
  LoginResponse,
  NetworkInterface,
  RouteFileInfo,
  RouteRuleSet,
  SSESnapshot,
  ServerStatus,
  StatsSnapshot
} from '../types'

type EventDisposer = () => void

function detectWails(): boolean {
  try {
    if (typeof window === 'undefined') return false
    const w = window as any
    return !!(w.go && w.go.main && w.runtime)
  } catch {
    return false
  }
}

let wailsModule: typeof import('./api-wails') | null = null
let httpModule: typeof import('./api-http') | null = null

async function getWails() {
  if (!wailsModule) wailsModule = await import('./api-wails')
  return wailsModule
}

async function getHttp() {
  if (!httpModule) httpModule = await import('./api-http')
  return httpModule
}

export function isWails(): boolean {
  return detectWails()
}

export function isWebLoggedIn(): boolean {
  if (detectWails()) {
    localStorage.removeItem('goproxy_token')
    return true
  }
  return !!localStorage.getItem('goproxy_token')
}

export async function webLogin(username: string, password: string): Promise<LoginResponse> {
  if (detectWails()) return (await getWails()).webLogin(username, password)
  return (await getHttp()).webLogin(username, password)
}

export async function webCheckAuth(): Promise<CheckAuthResponse> {
  if (detectWails()) return { valid: true, username: '', mustChangePwd: false }
  return (await getHttp()).webCheckAuth()
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<ChangePasswordResponse> {
  return (await getHttp()).changePassword(oldPassword, newPassword)
}

export async function getConfig(): Promise<AppConfig> {
  return detectWails() ? (await getWails()).getConfig() : (await getHttp()).getConfig()
}

export async function getLocalIPAddresses(): Promise<string[]> {
  return detectWails() ? (await getWails()).getLocalIPAddresses() : (await getHttp()).getLocalIPAddresses()
}

export async function getNetworkInterfaces(): Promise<NetworkInterface[]> {
  return detectWails() ? (await getWails()).getNetworkInterfaces() : (await getHttp()).getNetworkInterfaces()
}

export async function saveConfig(config: AppConfig): Promise<void> {
  return detectWails() ? (await getWails()).saveConfig(config) : (await getHttp()).saveConfig(config)
}

export async function listRouteFiles(): Promise<RouteFileInfo[]> {
  return detectWails() ? (await getWails()).listRouteFiles() : (await getHttp()).listRouteFiles()
}

export async function loadRouteFile(name: string): Promise<RouteRuleSet> {
  return detectWails() ? (await getWails()).loadRouteFile(name) : (await getHttp()).loadRouteFile(name)
}

export async function saveRouteFile(name: string, ruleSet: RouteRuleSet): Promise<void> {
  return detectWails() ? (await getWails()).saveRouteFile(name, ruleSet) : (await getHttp()).saveRouteFile(name, ruleSet)
}

export async function createRouteFile(name: string): Promise<void> {
  return detectWails() ? (await getWails()).createRouteFile(name) : (await getHttp()).createRouteFile(name)
}

export async function deleteRouteFile(name: string): Promise<void> {
  return detectWails() ? (await getWails()).deleteRouteFile(name) : (await getHttp()).deleteRouteFile(name)
}

export async function setActiveRouteFile(name: string): Promise<void> {
  return detectWails() ? (await getWails()).setActiveRouteFile(name) : (await getHttp()).setActiveRouteFile(name)
}

export async function startServer() {
  return detectWails() ? (await getWails()).startServer() : (await getHttp()).startServer()
}

export async function stopServer() {
  return detectWails() ? (await getWails()).stopServer() : (await getHttp()).stopServer()
}

export async function getServerStatus(): Promise<ServerStatus> {
  return detectWails() ? (await getWails()).getServerStatus() : (await getHttp()).getServerStatus()
}

export async function getStats(): Promise<StatsSnapshot> {
  return detectWails() ? (await getWails()).getStats() : (await getHttp()).getStats()
}

export async function getActiveConnections(): Promise<ActiveConnection[]> {
  return detectWails() ? (await getWails()).getActiveConnections() : (await getHttp()).getActiveConnections()
}

export async function getRecentLogs(n: number): Promise<LogEntry[]> {
  return detectWails() ? (await getWails()).getRecentLogs(n) : (await getHttp()).getRecentLogs(n)
}

export async function clearLogs(): Promise<void> {
  return detectWails() ? (await getWails()).clearLogs() : (await getHttp()).clearLogs()
}

export async function setAuthEnabled(enabled: boolean): Promise<void> {
  return detectWails() ? (await getWails()).setAuthEnabled(enabled) : (await getHttp()).setAuthEnabled(enabled)
}

export async function addUser(username: string, password: string): Promise<void> {
  return detectWails() ? (await getWails()).addUser(username, password) : (await getHttp()).addUser(username, password)
}

export async function removeUser(username: string): Promise<void> {
  return detectWails() ? (await getWails()).removeUser(username) : (await getHttp()).removeUser(username)
}

export async function resetUserPassword(username: string, password: string): Promise<void> {
  return detectWails() ? (await getWails()).resetUserPassword(username, password) : (await getHttp()).resetUserPassword(username, password)
}

export async function getTrayState() {
  return detectWails() ? (await getWails()).getTrayState() : (await getHttp()).getTrayState()
}

export async function showWindow(): Promise<void> {
  return detectWails() ? (await getWails()).showWindow() : (await getHttp()).showWindow()
}

export async function hideToTray(): Promise<void> {
  return detectWails() ? (await getWails()).hideToTray() : (await getHttp()).hideToTray()
}

export async function quitApp(): Promise<void> {
  return detectWails() ? (await getWails()).quitApp() : (await getHttp()).quitApp()
}

export async function copyText(text: string): Promise<void> {
  return detectWails() ? (await getWails()).copyText(text) : (await getHttp()).copyText(text)
}

export function onEvent<T>(eventName: string, callback: (payload: T) => void): EventDisposer {
  if (detectWails()) {
    getWails().then((m) => m.onEvent(eventName, callback))
    return () => {}
  }
  let disposer: EventDisposer = () => {}
  getHttp().then((m) => {
    disposer = m.onEvent(eventName, callback)
  })
  return () => disposer()
}

export function onServerSnapshot(callback: (snapshot: SSESnapshot) => void): EventDisposer {
  if (detectWails()) return () => {}
  let disposer: EventDisposer = () => {}
  getHttp().then((m) => {
    disposer = m.onServerSnapshot(callback)
  })
  return () => disposer()
}

export { type EventDisposer }
export { isWebLoggedIn as checkWebLoggedIn } from './api-http'
