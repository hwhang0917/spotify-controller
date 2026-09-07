<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import * as api from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import type { Config, Device, ServerStatus, SourceStatus, State } from './types'
import { fmtDuration } from './types'

const cfg = ref<Config | null>(null)
const sources = ref<SourceStatus[]>([])
const server = ref<ServerStatus>({ running: false, port: 0, url: '' })
const state = ref<State | null>(null)
const devices = ref<Device[]>([])
const foldersText = ref('')
const busy = ref('')
const error = ref('')
const notice = ref('')

async function run(label: string, fn: () => Promise<unknown>) {
  busy.value = label
  error.value = ''
  notice.value = ''
  try {
    await fn()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = ''
    await refresh()
  }
}

async function refresh() {
  const [c, s, st, ps] = await Promise.all([api.GetConfig(), api.Sources(), api.Status(), api.GetState()])
  cfg.value = c
  sources.value = s
  server.value = st
  state.value = ps
  foldersText.value = c.local.folders.join('\n')
}

function saveConfig() {
  if (!cfg.value) return
  const c = { ...cfg.value, local: { folders: foldersText.value.split('\n').map((l) => l.trim()).filter(Boolean) } }
  return run('save', () => api.SaveConfig(c))
}

const toggleServer = () =>
  run('server', () => (server.value.running ? api.StopServer() : api.StartServer(server.value.port)))

const useSource = (id: string) => run(id, () => api.SetActiveSource(id))

const rescan = () =>
  run('rescan', async () => {
    const n = await api.LocalRescan()
    notice.value = `${n} track(s) indexed`
  })

const connect = () =>
  run('connect', async () => {
    await saveConfig()
    await api.SpotifyConnect()
    notice.value = 'Spotify connected'
  })

const loadDevices = () => run('devices', async () => { devices.value = await api.SpotifyDevices() })

const togglePlay = () => run('play', () => (state.value?.nowPlaying?.playing ? api.Pause() : api.Resume()))

let stop: (() => void) | null = null
onMounted(async () => {
  await refresh()
  stop = EventsOn('state', (s: State) => { state.value = s })
})
onUnmounted(() => stop?.())
</script>

<template>
  <main class="min-h-screen p-8 space-y-6 max-w-5xl">
    <header class="flex items-end justify-between">
      <div>
        <p class="eyebrow">admin</p>
        <h1 class="text-2xl font-semibold tracking-tight">vibe-music</h1>
      </div>
      <p v-if="error" class="text-sm text-error">{{ error }}</p>
      <p v-else-if="notice" class="text-sm text-link">{{ notice }}</p>
    </header>

    <div v-if="cfg" class="grid gap-6 md:grid-cols-2">
      <!-- Server -->
      <section class="card space-y-4">
        <p class="eyebrow">server</p>
        <label class="block text-sm text-body">
          Guest port
          <input v-model.number="server.port" type="number" :disabled="server.running" class="input mt-1" />
        </label>
        <div class="flex items-center gap-3">
          <button @click="toggleServer" :disabled="!!busy" :class="server.running ? 'btn-ghost' : 'btn-primary'">
            {{ server.running ? 'Stop server' : 'Start server' }}
          </button>
          <span v-if="server.running" class="text-sm text-body">
            <span class="font-mono text-ink select-all">{{ server.url }}</span>
            · {{ state?.guests ?? 0 }} guest(s)
          </span>
        </div>
        <label class="block text-sm text-body">
          Skip when this share of guests vote
          <div class="flex items-center gap-3 mt-1">
            <input v-model.number="cfg.skipRatio" type="range" min="0.1" max="1" step="0.1" class="flex-1" @change="saveConfig" />
            <span class="font-mono text-sm w-12 text-right">{{ Math.round(cfg.skipRatio * 100) }}%</span>
          </div>
        </label>
      </section>

      <!-- Now playing -->
      <section class="card space-y-4">
        <p class="eyebrow">now playing</p>
        <div v-if="state?.nowPlaying" class="flex gap-4">
          <img v-if="state.nowPlaying.track.artworkUrl?.startsWith('http')" :src="state.nowPlaying.track.artworkUrl" class="w-20 h-20 rounded-sm object-cover" alt="" />
          <div v-else class="w-20 h-20 rounded-sm bg-hairline-soft" />
          <div class="min-w-0">
            <p class="font-medium truncate">{{ state.nowPlaying.track.title }}</p>
            <p class="text-sm text-body truncate">{{ state.nowPlaying.track.artist || '—' }}</p>
            <p class="text-xs text-mute mt-1">
              {{ fmtDuration(state.nowPlaying.track.duration) }}
              <span v-if="state.nowPlaying.requestedBy"> · requested by {{ state.nowPlaying.requestedBy }}</span>
              · skip votes {{ state.skipVotes }}/{{ state.skipThreshold }}
            </p>
          </div>
        </div>
        <p v-else class="text-sm text-mute">Nothing playing. Guests can request a song.</p>
        <div class="flex items-center gap-3">
          <button class="btn-ghost" :disabled="!state?.nowPlaying || !!busy" @click="togglePlay">
            {{ state?.nowPlaying?.playing ? 'Pause' : 'Resume' }}
          </button>
          <button class="btn-ghost" :disabled="!state?.nowPlaying || !!busy" @click="run('skip', api.Skip)">Skip</button>
          <input type="range" min="0" max="100" :value="state?.volume ?? 100" class="flex-1" @change="(e) => run('volume', () => api.SetVolume(Number((e.target as HTMLInputElement).value)))" />
        </div>
        <ol v-if="state?.queue.length" class="divide-y divide-hairline text-sm">
          <li v-for="(it, i) in state.queue" :key="it.id" class="py-2 flex justify-between gap-3">
            <span class="truncate"><span class="font-mono text-mute mr-2">{{ i + 1 }}</span>{{ it.track.title }} <span class="text-mute">· {{ it.requestedBy }}</span></span>
            <span class="font-mono text-mute shrink-0">▲ {{ it.votes }}</span>
          </li>
        </ol>
      </section>

      <!-- Sources -->
      <section class="card space-y-4 md:col-span-2">
        <p class="eyebrow">source · one active at a time</p>
        <div class="grid gap-3 sm:grid-cols-2">
          <button
            v-for="s in sources" :key="s.id"
            @click="useSource(s.id)" :disabled="s.active || !s.ready || !!busy"
            class="text-left rounded-md border p-4 transition-colors disabled:cursor-default"
            :class="s.active ? 'border-ink' : 'border-hairline hover:border-body'"
          >
            <div class="flex items-center justify-between">
              <span class="font-medium">{{ s.name }}</span>
              <span class="eyebrow" :class="s.active ? 'text-link' : ''">{{ s.active ? 'active' : s.ready ? 'ready' : 'setup needed' }}</span>
            </div>
            <p class="text-sm text-mute mt-1">{{ s.detail }}</p>
          </button>
        </div>

        <div class="grid gap-6 md:grid-cols-2 pt-2">
          <div class="space-y-3">
            <p class="eyebrow">local files</p>
            <label class="block text-sm text-body">
              Folders, one per line
              <textarea v-model="foldersText" rows="4" class="input mt-1 h-auto py-2 font-mono text-xs" placeholder="C:\Users\me\Music" />
            </label>
            <div class="flex gap-3">
              <button class="btn-primary" :disabled="!!busy" @click="saveConfig">Save</button>
              <button class="btn-ghost" :disabled="!!busy" @click="rescan">Rescan</button>
            </div>
          </div>

          <div class="space-y-3">
            <p class="eyebrow">spotify</p>
            <label class="block text-sm text-body">
              Client ID <span class="text-mute">(your own app at developer.spotify.com)</span>
              <input v-model="cfg.spotify.clientId" class="input mt-1 font-mono text-xs" />
            </label>
            <div class="flex gap-3">
              <button v-if="!sources.find((s) => s.id === 'spotify')?.ready" class="btn-primary" :disabled="!cfg.spotify.clientId || !!busy" @click="connect">
                {{ busy === 'connect' ? 'Waiting for browser…' : 'Connect' }}
              </button>
              <button v-else class="btn-ghost" :disabled="!!busy" @click="run('disconnect', api.SpotifyDisconnect)">Disconnect</button>
              <button class="btn-ghost" :disabled="!!busy" @click="loadDevices">Devices</button>
            </div>
            <label v-if="devices.length" class="block text-sm text-body">
              Playback device
              <select v-model="cfg.spotify.deviceId" class="input mt-1" @change="saveConfig">
                <option value="">Whatever is active</option>
                <option v-for="d in devices" :key="d.id" :value="d.id">{{ d.name }} · {{ d.type }}{{ d.active ? ' · active' : '' }}</option>
              </select>
            </label>
            <p class="text-xs text-mute">
              Redirect URI to register: <span class="font-mono text-ink select-all">http://127.0.0.1/callback</span>
            </p>
          </div>
        </div>
      </section>
    </div>
  </main>
</template>
