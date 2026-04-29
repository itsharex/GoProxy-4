import type { ActiveConnection, AppConfig, LogEntry, ServerStatus, StatsSnapshot } from '../types'
import {
  AddUser,
  GetActiveConnections,
  GetConfig,
  GetLocalIPAddresses,
  GetRecentLogs,
  GetServerStatus,
  GetStats,
  GetTrayState,
  HideToTray,
  QuitApp,
  RemoveUser,
  ResetUserPassword,
  SaveConfig,
  SetAuthEnabled,
  ShowWindow,
  StartServer,
  StopServer
} from '../../wailsjs/go/main/App'
import { ClipboardSetText, EventsOn } from '../../wailsjs/runtime/runtime'

type EventDisposer = () => void

export function getConfig() {
  return GetConfig() as unknown as Promise<AppConfig>
}

export function getLocalIPAddresses() {
  return GetLocalIPAddresses() as unknown as Promise<string[]>
}

export function saveConfig(config: AppConfig) {
  return SaveConfig(config as unknown as Parameters<typeof SaveConfig>[0])
}

export function startServer() {
  return StartServer()
}

export function stopServer() {
  return StopServer()
}

export function getServerStatus() {
  return GetServerStatus() as unknown as Promise<ServerStatus>
}

export function getStats() {
  return GetStats() as unknown as Promise<StatsSnapshot>
}

export function getActiveConnections() {
  return GetActiveConnections() as unknown as Promise<ActiveConnection[]>
}

export function getRecentLogs(n: number) {
  return GetRecentLogs(n) as unknown as Promise<LogEntry[]>
}

export function setAuthEnabled(enabled: boolean) {
  return SetAuthEnabled(enabled)
}

export function addUser(username: string, password: string) {
  return AddUser(username, password)
}

export function removeUser(username: string) {
  return RemoveUser(username)
}

export function resetUserPassword(username: string, password: string) {
  return ResetUserPassword(username, password)
}

export function getTrayState() {
  return GetTrayState()
}

export function showWindow() {
  return ShowWindow()
}

export function hideToTray() {
  return HideToTray()
}

export function quitApp() {
  return QuitApp()
}

export function copyText(text: string) {
  return ClipboardSetText(text)
}

export function onEvent<T>(eventName: string, callback: (payload: T) => void): EventDisposer {
  try {
    return EventsOn(eventName, (payload: T) => callback(payload))
  } catch {
    return () => undefined
  }
}
