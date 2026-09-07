<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ChevronUp, Plus, Search, Users, X } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import NowPlaying from './NowPlaying.vue'
import TrackRow from './TrackRow.vue'
import LocaleToggle from './LocaleToggle.vue'
import SourceIcon from './SourceIcon.vue'
import { toast } from 'vue-sonner'
import { Toaster } from '@/components/ui/sonner'
import { locale, t, tError } from './i18n'
import { fmtDuration } from './types'
import type { State, Track } from './types'

const name = ref('')
const nameInput = ref('')
// Until /api/me answers we do not know whether to show the gate or the player.
const loading = ref(true)
const state = ref<State | null>(null)
const query = ref('')
const results = ref<Track[]>([])
const searching = ref(false)
const top = ref<Track[]>([])

// Source chips: guests search one enabled source at a time. The selection
// follows what is enabled; disabled sources stay visible but inert.
const sources = computed(() => state.value?.sources ?? [])
const enabledSources = computed(() => sources.value.filter((s) => s.enabled))
const chosen = ref('')
const selected = computed(() => enabledSources.value.find((s) => s.id === chosen.value)?.id ?? enabledSources.value[0]?.id ?? '')
const canSearch = computed(() => selected.value !== '')
watch(selected, (id, prev) => { if (id !== prev) { results.value = []; loadTop() } })
function pick(id: string) {
  if (!enabledSources.value.some((s) => s.id === id)) return
  chosen.value = id
  if (query.value.trim()) search()
}

// Most played on the selected source; refreshed when it or the track changes.
async function loadTop() {
  if (!selected.value) { top.value = []; return }
  try { top.value = await api<Track[]>('GET', `/api/top?source=${encodeURIComponent(selected.value)}`) } catch { /* keep the old list */ }
}

// Browse mode when the search box is empty: what's popular here, or the source's chart.
const browse = ref<'top' | 'chart'>('top')
const chart = ref<Track[]>([])
const chartLoading = ref(false)
const hasChart = computed(() => enabledSources.value.find((s) => s.id === selected.value)?.hasChart ?? false)
const region = computed(() => (locale.value === 'ko' ? 'KR' : (navigator.language.split('-')[1] ?? 'US').toUpperCase()))
async function loadChart() {
  if (!selected.value || !hasChart.value) { chart.value = []; return }
  chartLoading.value = true
  try { chart.value = await api<Track[]>('GET', `/api/chart?source=${encodeURIComponent(selected.value)}&region=${region.value}`) }
  catch (e) { toast.error(tError((e as Error).message)) }
  finally { chartLoading.value = false }
}
watch([browse, selected, hasChart], () => { if (browse.value === 'chart') loadChart() })
const error = ref('')
const connected = ref(false)
const blocked = ref(false)
const inviteRequired = ref(false)
// Waiting screens (invite required, blocked) have no event stream, so they
// poll /api/me until the host lets the guest in, then continue in place.
const ACCESS_MS = 3000
let access: number | undefined
function waitForAccess() {
  if (access !== undefined) return
  access = window.setInterval(async () => {
    try {
      const me = await api<{ name: string }>('GET', '/api/me')
      window.clearInterval(access)
      access = undefined
      inviteRequired.value = false
      blocked.value = false
      name.value = me.name
      startStream()
    } catch { /* still waiting */ }
  }, ACCESS_MS)
}
const inviteInvalid = new URLSearchParams(location.search).get('invite') === 'invalid'
// Health check: the stream dropping could be a kick or a reconnect blip, so
// the page only goes "offline" once /api/health itself fails. It then probes
// until the server answers again.
const HEALTH_MS = 3000
const offline = ref(false)
let health: number | undefined

async function probe() {
  try {
    const res = await fetch('/api/health', { cache: 'no-store' })
    if (!res.ok) throw new Error(String(res.status))
    if (offline.value) {
      offline.value = false
      window.clearInterval(health)
      health = undefined
      await act(() => api('GET', '/api/me')) // re-check block / invite status after the host came back
    }
  } catch {
    offline.value = true
    connected.value = false
    if (health === undefined) health = window.setInterval(probe, HEALTH_MS)
  }
}

class ApiError extends Error {
  constructor(public code: string, public status: number) { super(code) }
}

async function api<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(data.error ?? res.statusText, res.status)
  return data as T
}

// Actions toast their outcome: red on failure, green when fn returns a message.
async function act(fn: () => Promise<string | void>) {
  error.value = ''
  try {
    const msg = await fn()
    if (msg) toast.success(msg)
  } catch (e) {
    if (e instanceof ApiError && e.code === 'blocked') { blocked.value = true; es?.close(); es = null; waitForAccess(); return }
    if (e instanceof ApiError && e.code === 'invite_required') { inviteRequired.value = true; waitForAccess(); return }
    if (!(e instanceof ApiError)) { probe(); return } // network failure: is the server gone?
    const msg = tError((e as Error).message)
    if (name.value) toast.error(msg)
    else error.value = msg // name gate shows it inline
  }
}

// Host actions arrive as a one-shot event on a state frame.
function announce(ev: State['event']) {
  if (!ev) return
  toast.info(t(`toast.${ev.type}`, { title: ev.title ?? '', pos: fmtDuration(ev.position ?? 0) }))
}

const saveName = () => act(async () => {
  const me = await api<{ name: string }>('POST', '/api/me', { name: nameInput.value })
  name.value = me.name
})

// Search runs on Enter or the button, not on every keystroke: YouTube
// searches cost quota and Spotify rate-limits per app.
function search() {
  const q = query.value.trim()
  if (!q || !canSearch.value) { results.value = []; return }
  act(async () => {
    searching.value = true
    try { results.value = await api<Track[]>('GET', `/api/search?q=${encodeURIComponent(q)}&source=${encodeURIComponent(selected.value)}`) }
    finally { searching.value = false }
  })
}
function onQuery() {
  if (!query.value.trim()) results.value = []
}

const request = (tr: Track) => act(async () => {
  await api('POST', '/api/queue', tr)
  query.value = ''
  results.value = []
  return t('toast.requested', { title: tr.title })
})
const vote = (id: string) => act(async () => { await api('POST', `/api/queue/${id}/vote`); return t('toast.voted') })
const remove = (id: string) => act(async () => { await api('DELETE', `/api/queue/${id}`); return t('toast.removed') })
const voteSkip = () => act(async () => { await api('POST', '/api/skip'); return t('toast.skipVoted') })

const isSpotify = computed(() => enabledSources.value.some((s) => s.id === 'spotify'))
const isYouTube = computed(() => enabledSources.value.some((s) => s.id === 'youtube'))

let es: EventSource | null = null
function startStream() {
  if (es) return
  es = new EventSource('/api/events')
  es.addEventListener('state', (e) => {
    const s: State = JSON.parse((e as MessageEvent).data)
    const changed = s.nowPlaying?.track.id !== state.value?.nowPlaying?.track.id
    state.value = s
    announce(s.event)
    if (changed) loadTop()
  })
  es.onopen = () => { connected.value = true }
  es.onerror = () => {
    connected.value = false
    // A kick just reconnects; a block or a dead server shows up on the probe.
    probe()
    act(() => api('GET', '/api/me'))
  }
}

onMounted(async () => {
  await act(async () => {
    const me = await api<{ name: string }>('GET', '/api/me')
    name.value = me.name
  })
  loading.value = false
  if (blocked.value || inviteRequired.value || offline.value) return
  startStream()
})
onUnmounted(() => { es?.close(); window.clearInterval(health); window.clearInterval(access) })
</script>

<template>
  <div v-if="offline" class="fixed inset-x-0 top-0 z-50 bg-destructive px-4 py-2 text-center text-sm font-medium text-white shadow-md">
    {{ t('offline.ribbon') }}
  </div>
  <Toaster position="bottom-right" rich-colors close-button />

  <!-- deciding which screen applies: keep the canvas blank rather than flash the gate -->
  <main v-if="loading && !offline" class="min-h-screen" aria-busy="true" />

  <!-- invitation required -->
  <main v-else-if="inviteRequired && !blocked" class="min-h-screen flex items-center justify-center p-6" :class="offline ? 'pointer-events-none opacity-50' : ''">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <p class="eyebrow">vibe-music</p>
        <CardTitle class="text-xl">{{ t('invite.title') }}</CardTitle>
      </CardHeader>
      <CardContent class="space-y-2 text-sm text-body">
        <p>{{ t('invite.body') }}</p>
        <p v-if="inviteInvalid" class="text-destructive">{{ t('invite.invalid') }}</p>
      </CardContent>
    </Card>
  </main>

  <!-- blocked -->
  <main v-else-if="blocked" class="min-h-screen flex items-center justify-center p-6">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <p class="eyebrow">vibe-music</p>
        <CardTitle class="text-xl">{{ t('blocked.title') }}</CardTitle>
      </CardHeader>
      <CardContent class="text-sm text-body">{{ t('blocked.body') }}</CardContent>
    </Card>
  </main>

  <!-- name gate -->
  <main v-else-if="!name" class="min-h-screen flex items-center justify-center p-6" :class="offline ? 'pointer-events-none opacity-50' : ''">
    <Card class="w-full max-w-sm">
      <CardHeader class="flex-row items-center justify-between">
        <p class="eyebrow">vibe-music</p>
        <LocaleToggle />
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="saveName">
          <h1 class="text-2xl font-semibold tracking-tight">{{ t('gate.title') }}</h1>
          <Input v-model="nameInput" :placeholder="t('gate.placeholder')" maxlength="24" autofocus />
          <Button type="submit" class="w-full" :disabled="!nameInput.trim()">{{ t('gate.join') }}</Button>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
        </form>
      </CardContent>
    </Card>
  </main>

  <!-- player -->
  <main v-else class="min-h-screen mx-auto max-w-3xl space-y-4 p-4 sm:p-8" :class="offline ? 'pointer-events-none select-none opacity-50' : ''" :aria-disabled="offline">
    <header class="flex items-center justify-between">
      <div>
        <p class="eyebrow">vibe-music</p>
        <h1 class="text-xl font-semibold tracking-tight">{{ t('header.hi', { name }) }}</h1>
      </div>
      <div class="flex items-center gap-2">
        <Badge variant="outline" class="gap-1.5">
          <span class="size-1.5 rounded-full" :class="connected ? 'bg-link' : 'bg-destructive'" />
          <Users />
          {{ t('header.here', { n: state?.guests ?? 0 }) }}
        </Badge>
        <LocaleToggle />
      </div>
    </header>

    <NowPlaying :state="state" @skip="voteSkip" />

    <div class="grid gap-4 md:grid-cols-2">
      <!-- search -->
      <Card>
        <CardHeader>
          <p class="eyebrow">{{ t('search.eyebrow') }}</p>
          <div class="flex flex-wrap gap-1.5" role="tablist">
            <Button
              v-for="s in sources" :key="s.id" size="xs"
              :variant="s.id === selected ? 'default' : 'outline'"
              :disabled="!s.enabled" :title="s.enabled ? s.name : t('search.sourceOff', { name: s.name })"
              role="tab" :aria-selected="s.id === selected"
              @click="pick(s.id)"
            >
              <SourceIcon :source="s.id" class="size-3" />{{ s.name }}
            </Button>
          </div>
          <form class="flex gap-2" @submit.prevent="search">
            <div class="relative flex-1">
              <Search class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input v-model="query" type="search" enterkeyhint="search" class="pl-8" :disabled="!canSearch" :placeholder="canSearch ? t('search.placeholder') : t('search.noSource')" @input="onQuery" />
            </div>
            <Button type="submit" :disabled="!canSearch || !query.trim() || searching">{{ t('search.go') }}</Button>
          </form>
        </CardHeader>
        <CardContent>
          <div v-if="searching" class="space-y-3">
            <div v-for="i in 3" :key="i" class="flex items-center gap-3">
              <Skeleton class="size-11 rounded-md" />
              <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
            </div>
          </div>
          <div v-else-if="results.length" class="max-h-96 overflow-y-auto">
            <div class="divide-y">
              <TrackRow v-for="tr in results" :key="tr.id" :track="tr" :subtitle="tr.album">
                <Button size="sm" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
              </TrackRow>
            </div>
          </div>
          <p v-else-if="query" class="text-sm text-muted-foreground">{{ t('search.empty') }}</p>
          <template v-else-if="canSearch">
            <div class="mb-2 flex items-center gap-1.5">
              <Button size="xs" :variant="browse === 'top' ? 'secondary' : 'ghost'" @click="browse = 'top'">{{ t('search.top') }}</Button>
              <Button v-if="hasChart" size="xs" :variant="browse === 'chart' ? 'secondary' : 'ghost'" @click="browse = 'chart'">{{ t('search.chart', { n: chart.length || 50, region }) }}</Button>
            </div>
            <template v-if="browse === 'chart' && hasChart">
              <div v-if="chartLoading" class="space-y-3">
                <div v-for="i in 4" :key="i" class="flex items-center gap-3">
                  <Skeleton class="size-11 rounded-md" />
                  <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
                </div>
              </div>
              <div v-else-if="chart.length" class="max-h-96 overflow-y-auto">
                <div class="divide-y">
                  <TrackRow v-for="(tr, i) in chart" :key="tr.id" :track="tr" :index="i + 1" :subtitle="tr.album">
                    <Button size="sm" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
                  </TrackRow>
                </div>
              </div>
              <p v-else class="text-sm text-muted-foreground">{{ t('search.chartEmpty') }}</p>
            </template>
            <template v-else>
              <div v-if="top.length" class="max-h-96 overflow-y-auto">
                <div class="divide-y">
                  <TrackRow v-for="(tr, i) in top" :key="tr.id" :track="tr" :index="i + 1" :subtitle="tr.album">
                    <Button size="sm" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
                  </TrackRow>
                </div>
              </div>
              <p v-else class="text-sm text-muted-foreground">{{ t('search.hint') }}</p>
            </template>
          </template>
          <p v-else class="text-sm text-muted-foreground">{{ t('search.hint') }}</p>
        </CardContent>
      </Card>

      <!-- queue -->
      <Card>
        <CardHeader>
          <p class="eyebrow">{{ t('queue.eyebrow', { n: state?.queue.length ?? 0 }) }}</p>
        </CardHeader>
        <CardContent>
          <div v-if="state?.queue.length" class="max-h-96 overflow-y-auto">
            <div class="divide-y">
              <TrackRow v-for="(it, i) in state.queue" :key="it.id" :track="it.track" :index="i + 1" :subtitle="it.requestedBy">
                <Button variant="outline" size="sm" class="font-mono tabular-nums" @click="vote(it.id)">
                  <ChevronUp />{{ it.votes }}
                </Button>
                <Button v-if="it.mine" variant="ghost" size="icon-sm" :aria-label="t('queue.remove')" :title="t('queue.remove')" @click="remove(it.id)">
                  <X />
                </Button>
              </TrackRow>
            </div>
          </div>
          <p v-else class="text-sm text-muted-foreground">{{ t('queue.empty') }}</p>
        </CardContent>
      </Card>
    </div>

    <template v-if="isSpotify || isYouTube">
      <Separator />
      <footer class="pb-4 text-center text-xs text-muted-foreground">{{ t(isYouTube ? 'youtube.footer' : 'spotify.footer') }}</footer>
    </template>
  </main>
</template>
