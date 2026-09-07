// Mirrors the JSON shapes in app.go, internal/config and internal/player.
export interface Config {
  port: number
  activeSource: string
  skipRatio: number
  local: { folders: string[] }
  spotify: { clientId: string; deviceId?: string }
}

export interface SourceStatus {
  id: string
  name: string
  active: boolean
  ready: boolean
  detail: string
}

export interface Device {
  id: string
  name: string
  type: string
  active: boolean
}

export interface Track {
  id: string
  title: string
  artist: string
  album: string
  duration: number // nanoseconds (Go time.Duration)
  artworkUrl?: string
  externalUrl?: string
}

export interface QueueItem {
  id: string
  track: Track
  requestedBy: string
  requestedAt: string
  votes: number
}

export interface State {
  source: { id: string; name: string } | null
  nowPlaying: {
    track: Track
    playing: boolean
    position: number
    at: string
    requestedBy?: string
  } | null
  queue: QueueItem[]
  skipVotes: number
  skipThreshold: number
  volume: number
  guests: number
}

export interface ServerStatus {
  running: boolean
  port: number
  url: string
}

export const NS_PER_SEC = 1e9

export function fmtDuration(ns: number): string {
  if (!ns) return '--:--'
  const s = Math.round(ns / NS_PER_SEC)
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

export interface GuestInfo {
  id: string
  name: string
  connections: number
  blocked: boolean
  lastSeen: string
}
