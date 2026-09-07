<script setup lang="ts">
// The official YouTube IFrame player, framed from a loopback http page that Go
// serves (internal/source/youtube/player.html): YouTube refuses embeds without
// a Referer (error 153) and the wails:// origin on Linux/macOS sends none. Go
// drives it through the "yt:cmd" Wails event, relayed here as postMessage;
// the page reports state back the same way and this component forwards it to
// YouTubeReport. YouTube's terms require the player to stay visible, so it
// lives in the admin's player card whenever the YouTube source is active.
import { onMounted, onUnmounted, ref } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { YouTubePlayerURL, YouTubeReport } from '../wailsjs/go/main/App'

const props = defineProps<{ blockedLabel: string }>()

const frame = ref<HTMLIFrameElement | null>(null)
const src = ref('')
let origin = ''
let stop: (() => void) | undefined

function onMessage(e: MessageEvent) {
  if (e.origin !== origin || e.source !== frame.value?.contentWindow) return
  YouTubeReport(e.data).catch(() => {})
}

onMounted(async () => {
  origin = await YouTubePlayerURL()
  src.value = `${origin}/?blocked=${encodeURIComponent(props.blockedLabel)}`
  window.addEventListener('message', onMessage)
  stop = EventsOn('yt:cmd', (c: unknown) => frame.value?.contentWindow?.postMessage(c, origin))
})
onUnmounted(() => {
  stop?.()
  window.removeEventListener('message', onMessage)
  YouTubeReport({ ready: false, videoId: '', state: -1, position: 0, duration: 0 }).catch(() => {})
})
</script>

<template>
  <div class="relative aspect-video w-full bg-black">
    <iframe v-if="src" ref="frame" :src="src" class="size-full border-0" allow="autoplay; encrypted-media; fullscreen" />
  </div>
</template>
