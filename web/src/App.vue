<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import SpotifyMark from './SpotifyMark.vue'
import type { State, Track } from './types'
import { fmtDuration, NS_PER_MS } from './types'

const SEARCH_DEBOUNCE_MS = 300

const name = ref('')
const nameInput = ref('')
const state = ref<State | null>(null)
const query = ref('')
const results = ref<Track[]>([])
const searching = ref(false)
const error = ref('')
const now = ref(Date.now())
const connected = ref(false)

async function api<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(data.error ?? res.statusText)
  return data as T
}

async function act(fn: () => Promise<unknown>) {
  error.value = ''
  try {
    await fn()
  } catch (e) {
    error.value = (e as Error).message
  }
}

const saveName = () => act(async () => {
  const me = await api<{ name: string }>('POST', '/api/me', { name: nameInput.value })
  name.value = me.name
})

let timer: number | undefined
function onQuery() {
  window.clearTimeout(timer)
  const q = query.value.trim()
  if (!q) { results.value = []; return }
  timer = window.setTimeout(() => act(async () => {
    searching.value = true
    try { results.value = await api<Track[]>('GET', `/api/search?q=${encodeURIComponent(q)}`) }
    finally { searching.value = false }
  }), SEARCH_DEBOUNCE_MS)
}

const request = (t: Track) => act(async () => {
  await api('POST', '/api/queue', t)
  query.value = ''
  results.value = []
})
const vote = (id: string) => act(() => api('POST', `/api/queue/${id}/vote`))
const voteSkip = () => act(() => api('POST', '/api/skip'))

// Position interpolates from the last frame; the server only pushes on change.
const position = computed(() => {
  const np = state.value?.nowPlaying
  if (!np) return 0
  const base = np.position / NS_PER_MS
  return np.playing ? base + (now.value - new Date(np.at).getTime()) : base
})
const progressPct = computed(() => {
  const d = state.value?.nowPlaying?.track.duration
  return d ? Math.min(100, (position.value * NS_PER_MS / d) * 100) : 0
})
const isSpotify = computed(() => state.value?.source?.id === 'spotify')

let es: EventSource | null = null
let tick: number | undefined
onMounted(async () => {
  const me = await api<{ name: string }>('GET', '/api/me')
  name.value = me.name
  es = new EventSource('/api/events')
  es.addEventListener('state', (e) => { state.value = JSON.parse((e as MessageEvent).data) })
  es.onopen = () => { connected.value = true }
  es.onerror = () => { connected.value = false }
  tick = window.setInterval(() => { now.value = Date.now() }, 1000)
})
onUnmounted(() => { es?.close(); window.clearInterval(tick) })
</script>

<template>
  <!-- name gate -->
  <main v-if="!name" class="min-h-screen flex items-center justify-center p-6">
    <form class="card w-full max-w-sm space-y-4" @submit.prevent="saveName">
      <p class="eyebrow">vibe-music</p>
      <h1 class="text-2xl font-semibold tracking-tight">What should we call you?</h1>
      <input v-model="nameInput" class="input" placeholder="Your name" maxlength="24" autofocus />
      <button class="btn-primary w-full" :disabled="!nameInput.trim()">Join</button>
      <p v-if="error" class="text-sm text-error">{{ error }}</p>
    </form>
  </main>

  <main v-else class="min-h-screen p-4 sm:p-8 space-y-4 max-w-2xl mx-auto">
    <header class="flex items-center justify-between">
      <div>
        <p class="eyebrow">vibe-music</p>
        <h1 class="text-xl font-semibold tracking-tight">Hi, {{ name }}</h1>
      </div>
      <p class="text-xs text-mute">
        <span :class="connected ? 'text-link' : 'text-error'">●</span>
        {{ state?.guests ?? 0 }} here
      </p>
    </header>
    <p v-if="error" class="text-sm text-error">{{ error }}</p>

    <!-- now playing -->
    <section class="card space-y-4">
      <p class="eyebrow">now playing<span v-if="state?.source"> · {{ state.source.name }}</span></p>
      <div v-if="state?.nowPlaying" class="space-y-4">
        <div class="flex gap-4">
          <img v-if="state.nowPlaying.track.artworkUrl" :src="state.nowPlaying.track.artworkUrl" class="w-24 h-24 rounded-sm object-cover shrink-0" alt="" />
          <div v-else class="w-24 h-24 rounded-sm bg-hairline-soft shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-lg font-semibold tracking-tight truncate">{{ state.nowPlaying.track.title }}</p>
            <p class="text-sm text-body truncate">{{ state.nowPlaying.track.artist || '—' }}</p>
            <p v-if="state.nowPlaying.requestedBy" class="text-xs text-mute mt-1">requested by {{ state.nowPlaying.requestedBy }}</p>
            <SpotifyMark v-if="state.nowPlaying.track.externalUrl" :href="state.nowPlaying.track.externalUrl" class="mt-2" />
          </div>
        </div>
        <div>
          <div class="h-1 rounded-full bg-hairline overflow-hidden">
            <div class="h-full bg-ink transition-[width]" :style="{ width: progressPct + '%' }" />
          </div>
          <div class="flex justify-between font-mono text-xs text-mute mt-1">
            <span>{{ fmtDuration(position * NS_PER_MS) }}</span>
            <span>{{ state.nowPlaying.playing ? '' : 'paused · ' }}{{ fmtDuration(state.nowPlaying.track.duration) }}</span>
          </div>
        </div>
        <button class="btn-ghost w-full" @click="voteSkip">
          Vote to skip · {{ state.skipVotes }}/{{ state.skipThreshold }}
        </button>
      </div>
      <p v-else class="text-sm text-mute">Nothing playing. Request something below.</p>
    </section>

    <!-- search -->
    <section class="card space-y-3">
      <p class="eyebrow">request a song</p>
      <input v-model="query" class="input" placeholder="Search title, artist, album" @input="onQuery" />
      <ul v-if="results.length" class="divide-y divide-hairline">
        <li v-for="t in results" :key="t.id" class="py-2 flex items-center gap-3">
          <img v-if="t.artworkUrl" :src="t.artworkUrl" class="w-10 h-10 rounded-sm object-cover shrink-0" alt="" />
          <div v-else class="w-10 h-10 rounded-sm bg-hairline-soft shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium truncate">{{ t.title }}</p>
            <p class="text-xs text-body truncate">{{ t.artist }}<span v-if="t.album"> · {{ t.album }}</span></p>
            <SpotifyMark v-if="t.externalUrl" :href="t.externalUrl" />
          </div>
          <span class="font-mono text-xs text-mute">{{ fmtDuration(t.duration) }}</span>
          <button class="btn-primary" @click="request(t)">Request</button>
        </li>
      </ul>
      <p v-else-if="query && !searching" class="text-sm text-mute">No results.</p>
    </section>

    <!-- queue -->
    <section class="card space-y-3">
      <p class="eyebrow">up next · {{ state?.queue.length ?? 0 }}</p>
      <ol v-if="state?.queue.length" class="divide-y divide-hairline">
        <li v-for="(it, i) in state.queue" :key="it.id" class="py-2 flex items-center gap-3">
          <span class="font-mono text-xs text-mute w-5">{{ i + 1 }}</span>
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium truncate">{{ it.track.title }}</p>
            <p class="text-xs text-body truncate">{{ it.track.artist }} · {{ it.requestedBy }}</p>
            <SpotifyMark v-if="it.track.externalUrl" :href="it.track.externalUrl" />
          </div>
          <button class="btn-ghost font-mono" @click="vote(it.id)">▲ {{ it.votes }}</button>
        </li>
      </ol>
      <p v-else class="text-sm text-mute">Queue is empty.</p>
    </section>

    <footer v-if="isSpotify" class="text-xs text-mute text-center pb-4">
      Music plays on the host's Spotify account. Content provided by Spotify.
    </footer>
  </main>
</template>
