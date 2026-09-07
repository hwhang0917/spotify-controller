// Mirrors the JSON shapes served by internal/server and internal/player.
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
  mine?: boolean
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

export const NS_PER_MS = 1e6

export function fmtDuration(ns: number): string {
  if (!ns) return '--:--'
  const s = Math.round(ns / NS_PER_MS / 1000)
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}
