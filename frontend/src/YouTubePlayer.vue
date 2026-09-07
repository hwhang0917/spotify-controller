<script setup lang="ts">
// The official YouTube IFrame player. Go drives it through the "yt:cmd" Wails
// event; this component reports state back with YouTubeReport. YouTube's
// terms require the player to stay visible, so it lives in the admin's
// player card whenever the YouTube source is active.
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { YouTubeReport } from '../wailsjs/go/main/App'

const REPORT_MS = 500
const IFRAME_API = 'https://www.youtube.com/iframe_api'

type Cmd = { type: 'load' | 'play' | 'pause' | 'stop' | 'seek' | 'volume'; videoId?: string; seconds?: number; volume?: number }
// Minimal surface of YT.Player we use; the script is loaded at runtime.
interface YTPlayer {
  loadVideoById(id: string): void
  playVideo(): void
  pauseVideo(): void
  stopVideo(): void
  seekTo(s: number, allow: boolean): void
  setVolume(v: number): void
  getPlayerState(): number
  getCurrentTime(): number
  getDuration(): number
  getVideoData(): { video_id: string }
  destroy(): void
}
declare global {
  interface Window { YT?: { Player: new (el: HTMLElement, opts: unknown) => YTPlayer; PlayerState: Record<string, number> }; onYouTubeIframeAPIReady?: () => void }
}

const host = ref<HTMLDivElement | null>(null)
const ready = ref(false)
const blocked = ref(false) // autoplay refused: needs a click
let player: YTPlayer | null = null
let timer: number | undefined
let stop: (() => void) | undefined
let last = ''

function loadApi(): Promise<void> {
  if (window.YT?.Player) return Promise.resolve()
  return new Promise((resolve) => {
    window.onYouTubeIframeAPIReady = () => resolve()
    const s = document.createElement('script')
    s.src = IFRAME_API
    document.head.appendChild(s)
  })
}

function report() {
  if (!player || !ready.value) return
  const r = {
    ready: true,
    videoId: player.getVideoData?.().video_id ?? '',
    state: player.getPlayerState(),
    position: player.getCurrentTime() || 0,
    duration: player.getDuration() || 0,
  }
  // -1 unstarted / 5 cued after a load means autoplay was refused
  blocked.value = r.videoId !== '' && (r.state === -1 || r.state === 5)
  const key = JSON.stringify(r)
  if (key !== last) { last = key; YouTubeReport(r).catch(() => {}) }
}

// Autoplay refused: a real click on the overlay is a user gesture, which lifts the block.
function resume() { player?.playVideo() }

function run(c: Cmd) {
  if (!player) return
  switch (c.type) {
    case 'load': player.loadVideoById(c.videoId!); break
    case 'play': player.playVideo(); break
    case 'pause': player.pauseVideo(); break
    case 'stop': player.stopVideo(); break
    case 'seek': player.seekTo(c.seconds ?? 0, true); break
    case 'volume': player.setVolume(c.volume ?? 100); break
  }
}

onMounted(async () => {
  await loadApi()
  player = new window.YT!.Player(host.value!, {
    width: '100%', height: '100%',
    playerVars: { autoplay: 0, controls: 1, rel: 0, modestbranding: 1, playsinline: 1 },
    events: {
      onReady: () => { ready.value = true; report() },
      onStateChange: report,
    },
  })
  stop = EventsOn('yt:cmd', run)
  timer = window.setInterval(report, REPORT_MS)
})
onUnmounted(() => {
  stop?.()
  window.clearInterval(timer)
  YouTubeReport({ ready: false, videoId: '', state: -1, position: 0, duration: 0 }).catch(() => {})
  player?.destroy()
})
</script>

<template>
  <div class="relative aspect-video w-full bg-black">
    <div ref="host" class="size-full" />
    <button v-if="blocked" class="absolute inset-0 flex items-center justify-center bg-black/60 text-sm font-medium text-white" @click="resume">
      <slot name="blocked">Click to start playback</slot>
    </button>
  </div>
</template>
