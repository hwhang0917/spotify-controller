// Mirrors the JSON shapes in app.go, internal/config and internal/player.
export interface Config {
  port: number
  enabled: string[]
  skipRatio: number
  inviteOnly: boolean
  local: { folders: string[] }
  spotify: { clientId: string; deviceId?: string; callbackPort: number }
  youtube: { hasKey: boolean }
}

export interface SourceStatus {
  id: string
  name: string
  enabled: boolean
  wanted: boolean
  exclusive: boolean
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
  source: string
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
  sources: { id: string; name: string; enabled: boolean; exclusive: boolean }[]
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
  event?: { type: 'seek' | 'queue_moved' | 'queue_removed' | 'skipped'; title?: string; position?: number }
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
  admitted: boolean
  lastSeen: string
}

export interface Invitation {
  id: number
  label: string
  code?: string // only for codes created this session
  createdAt: string
  expiresAt: string
  revoked: boolean
  uses: number
}
