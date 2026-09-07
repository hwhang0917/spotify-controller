<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import * as api from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { locale, setLocale, t } from './i18n'
import type { Config, Device, GuestInfo, ServerStatus, SourceStatus, State } from './types'
import { fmtDuration } from './types'

const cfg = ref<Config | null>(null)
const sources = ref<SourceStatus[]>([])
const server = ref<ServerStatus>({ running: false, port: 0, url: '' })
const state = ref<State | null>(null)
const guests = ref<GuestInfo[]>([])
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
  const [c, s, st, ps, g] = await Promise.all([api.GetConfig(), api.Sources(), api.Status(), api.GetState(), api.Guests()])
  cfg.value = c
  sources.value = s
  server.value = st
  state.value = ps
  guests.value = g
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
const rescan = () => run('rescan', async () => { notice.value = t('local.indexed', { n: await api.LocalRescan() }) })
const connect = () => run('connect', async () => { await saveConfig(); await api.SpotifyConnect(); notice.value = t('spotify.connected') })
const loadDevices = () => run('devices', async () => { devices.value = await api.SpotifyDevices() })
const togglePlay = () => run('play', () => (state.value?.nowPlaying?.playing ? api.Pause() : api.Resume()))
const isReady = (id: string) => sources.value.find((s) => s.id === id)?.ready ?? false

// Wails events push changes; the poll is a safety net so the guest list and
// server status never go stale if a push is missed.
const POLL_MS = 3000

const stops: Array<() => void> = []
let poll: number | undefined
onMounted(async () => {
  try {
    await refresh()
  } catch (e) {
    error.value = String(e)
  }
  stops.push(EventsOn('state', (s: State) => { state.value = s }))
  stops.push(EventsOn('guests', (g: GuestInfo[]) => { guests.value = g }))
  poll = window.setInterval(async () => {
    try {
      const [g, st] = await Promise.all([api.Guests(), api.Status()])
      guests.value = g
      server.value = { ...st, port: server.value.port || st.port }
    } catch (e) {
      error.value = String(e)
    }
  }, POLL_MS)
})
onUnmounted(() => { stops.forEach((s) => s()); window.clearInterval(poll) })
</script>

<template>
  <main class="min-h-screen p-8 space-y-6 max-w-5xl mx-auto">
    <header class="flex items-end justify-between">
      <div>
        <p class="eyebrow">{{ t('admin') }}</p>
        <h1 class="text-2xl font-semibold tracking-tight">vibe-music</h1>
      </div>
      <div class="flex items-center gap-4">
        <p v-if="error" class="text-sm text-error">{{ error }}</p>
        <p v-else-if="notice" class="text-sm text-link">{{ notice }}</p>
        <button class="font-mono text-xs text-mute hover:text-ink" @click="setLocale(locale === 'en' ? 'ko' : 'en')">
          {{ locale === 'en' ? 'KO' : 'EN' }}
        </button>
      </div>
    </header>

    <div v-if="cfg" class="grid gap-6 md:grid-cols-2">
      <!-- Server -->
      <section class="card space-y-4">
        <p class="eyebrow">{{ t('server') }}</p>
        <label class="block text-sm text-body">
          {{ t('server.port') }}
          <input v-model.number="server.port" type="number" :disabled="server.running" class="input mt-1" />
        </label>
        <div class="flex items-center gap-3">
          <button @click="toggleServer" :disabled="!!busy" :class="server.running ? 'btn-ghost' : 'btn-primary'">
            {{ server.running ? t('server.stop') : t('server.start') }}
          </button>
          <span v-if="server.running" class="text-sm text-body">
            <span class="font-mono text-ink select-all">{{ server.url }}</span>
            · {{ t('server.guests', { n: state?.guests ?? 0 }) }}
          </span>
        </div>
        <label class="block text-sm text-body">
          {{ t('server.skipRatio') }}
          <div class="flex items-center gap-3 mt-1">
            <input v-model.number="cfg.skipRatio" type="range" min="0.1" max="1" step="0.1" class="flex-1" @change="saveConfig" />
            <span class="font-mono text-sm w-12 text-right">{{ Math.round(cfg.skipRatio * 100) }}%</span>
          </div>
        </label>
      </section>

      <!-- Now playing -->
      <section class="card space-y-4">
        <p class="eyebrow">{{ t('now') }}</p>
        <div v-if="state?.nowPlaying" class="flex gap-4">
          <img v-if="state.nowPlaying.track.artworkUrl?.startsWith('http')" :src="state.nowPlaying.track.artworkUrl" class="w-20 h-20 rounded-sm object-cover" alt="" />
          <div v-else class="w-20 h-20 rounded-sm bg-hairline-soft" />
          <div class="min-w-0">
            <p class="font-medium truncate">{{ state.nowPlaying.track.title }}</p>
            <p class="text-sm text-body truncate">{{ state.nowPlaying.track.artist || '—' }}</p>
            <p class="text-xs text-mute mt-1">
              {{ fmtDuration(state.nowPlaying.track.duration) }}
              <span v-if="state.nowPlaying.requestedBy"> · {{ t('now.requestedBy', { name: state.nowPlaying.requestedBy }) }}</span>
              · {{ t('now.skipVotes', { v: state.skipVotes, t: state.skipThreshold }) }}
            </p>
          </div>
        </div>
        <p v-else class="text-sm text-mute">{{ t('now.empty') }}</p>
        <div class="flex items-center gap-3">
          <button class="btn-ghost" :disabled="!state?.nowPlaying || !!busy" @click="togglePlay">
            {{ state?.nowPlaying?.playing ? t('now.pause') : t('now.resume') }}
          </button>
          <button class="btn-ghost" :disabled="!state?.nowPlaying || !!busy" @click="run('skip', api.Skip)">{{ t('now.skip') }}</button>
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
        <p class="eyebrow">{{ t('source') }}</p>
        <div class="grid gap-3 sm:grid-cols-2">
          <button
            v-for="s in sources" :key="s.id"
            @click="useSource(s.id)" :disabled="s.active || !s.ready || !!busy"
            class="text-left rounded-md border p-4 transition-colors disabled:cursor-default"
            :class="s.active ? 'border-ink' : 'border-hairline hover:border-body'"
          >
            <div class="flex items-center justify-between">
              <span class="font-medium">{{ s.name }}</span>
              <span class="eyebrow" :class="s.active ? 'text-link' : ''">{{ s.active ? t('source.active') : s.ready ? t('source.ready') : t('source.setup') }}</span>
            </div>
            <p class="text-sm text-mute mt-1">{{ s.detail }}</p>
          </button>
        </div>

        <div class="grid gap-6 md:grid-cols-2 pt-2">
          <div class="space-y-3">
            <p class="eyebrow">{{ t('local') }}</p>
            <label class="block text-sm text-body">
              {{ t('local.folders') }}
              <textarea v-model="foldersText" rows="4" class="input mt-1 h-auto py-2 font-mono text-xs" placeholder="C:\Users\me\Music" />
            </label>
            <div class="flex gap-3">
              <button class="btn-primary" :disabled="!!busy" @click="saveConfig">{{ t('local.save') }}</button>
              <button class="btn-ghost" :disabled="!!busy" @click="rescan">{{ t('local.rescan') }}</button>
            </div>
          </div>

          <div class="space-y-3">
            <p class="eyebrow">{{ t('spotify') }}</p>
            <label class="block text-sm text-body">
              {{ t('spotify.clientId') }} <span class="text-mute">{{ t('spotify.clientIdHint') }}</span>
              <input v-model="cfg.spotify.clientId" class="input mt-1 font-mono text-xs" />
            </label>
            <div class="flex gap-3">
              <button v-if="!isReady('spotify')" class="btn-primary" :disabled="!cfg.spotify.clientId || !!busy" @click="connect">
                {{ busy === 'connect' ? t('spotify.connecting') : t('spotify.connect') }}
              </button>
              <button v-else class="btn-ghost" :disabled="!!busy" @click="run('disconnect', api.SpotifyDisconnect)">{{ t('spotify.disconnect') }}</button>
              <button class="btn-ghost" :disabled="!!busy" @click="loadDevices">{{ t('spotify.devices') }}</button>
            </div>
            <label v-if="devices.length" class="block text-sm text-body">
              {{ t('spotify.device') }}
              <select v-model="cfg.spotify.deviceId" class="input mt-1" @change="saveConfig">
                <option value="">{{ t('spotify.deviceAny') }}</option>
                <option v-for="d in devices" :key="d.id" :value="d.id">{{ d.name }} · {{ d.type }}{{ d.active ? ' · ' + t('spotify.deviceActive') : '' }}</option>
              </select>
            </label>
            <p class="text-xs text-mute">
              {{ t('spotify.redirect') }} <span class="font-mono text-ink select-all">http://127.0.0.1/callback</span>
            </p>
          </div>
        </div>
      </section>

      <!-- Guests -->
      <section class="card space-y-3 md:col-span-2">
        <p class="eyebrow">{{ t('guests') }} · {{ guests.length }}</p>
        <p v-if="!guests.length" class="text-sm text-mute">{{ t('guests.empty') }}</p>
        <ul v-else class="divide-y divide-hairline">
          <li v-for="g in guests" :key="g.id" class="py-2 flex items-center gap-3 text-sm">
            <span :class="g.connections ? 'text-link' : 'text-faint'" :title="g.connections ? t('guests.online') : t('guests.offline')">●</span>
            <span class="min-w-0 flex-1 truncate">
              <span :class="g.name ? 'font-medium' : 'text-mute'">{{ g.name || t('guests.unnamed') }}</span>
              <span v-if="g.blocked" class="eyebrow text-error ml-2">{{ t('guests.blocked') }}</span>
              <span class="font-mono text-xs text-faint ml-2">{{ g.id.slice(0, 8) }}</span>
            </span>
            <button v-if="g.connections" class="btn-ghost" :disabled="!!busy" :title="t('guests.kickHint')" @click="run('kick', () => api.KickGuest(g.id))">{{ t('guests.kick') }}</button>
            <button v-if="!g.blocked" class="btn-ghost" :disabled="!!busy" :title="t('guests.removeHint')" @click="run('remove', () => api.RemoveGuest(g.id))">{{ t('guests.remove') }}</button>
            <button class="btn-ghost" :class="g.blocked ? '' : 'text-error'" :disabled="!!busy" :title="t('guests.blockHint')" @click="run('block', () => api.BlockGuest(g.id, !g.blocked))">
              {{ g.blocked ? t('guests.unblock') : t('guests.block') }}
            </button>
          </li>
        </ul>
      </section>
    </div>
  </main>
</template>
