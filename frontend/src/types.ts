export interface ProtocolConfig {
  enabled: boolean
  host: string
  port: number
}

export interface ServerConfig {
  socks5: ProtocolConfig
  http: ProtocolConfig
}

export interface RelayConfig {
  dialTimeoutSec: number
  readTimeoutSec: number
  maxConnections: number
  keepaliveSec: number
}

export interface LogConfig {
  level: 'debug' | 'info' | 'warn' | 'error'
  maxSizeMb: number
  maxBackups: number
  output: 'file' | 'console' | 'both'
}

export interface UIConfig {
  theme: 'light' | 'dark' | 'auto'
  language: string
  startMinimized: boolean
  showTrayIcon: boolean
}

export interface AppConfig {
  server: ServerConfig
  auth: AuthConfig
  relay: RelayConfig
  log: LogConfig
  ui: UIConfig
}

export interface AuthConfig {
  enabled: boolean
  users: AuthUser[]
}

export interface AuthUser {
  username: string
  password: string
}

export interface ServerStatus {
  running: boolean
  startedAt: string
  socks5Addr: string
  httpAddr: string
  activeConns: number
  totalConns: number
}

export interface StatsSnapshot {
  activeConns: number
  totalConns: number
  uploadBytes: number
  downloadBytes: number
  uploadRate: number
  downloadRate: number
  authFailures: number
}

export interface TrafficSample extends StatsSnapshot {
  time: string
}

export interface TrayState {
  enabled: boolean
  visible: boolean
  platform: string
  supportsMenu: boolean
  hideDescription: string
}

export interface ActiveConnection {
  id: number
  protocol: string
  clientAddr: string
  targetAddr: string
  uploadBytes: number
  downloadBytes: number
  openedAt: string
}

export interface LogEntry {
  time: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'
  message: string
  source: string
}
