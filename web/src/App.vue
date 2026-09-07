<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ChevronUp, Plus, Search, Users, X } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import NowPlaying from './NowPlaying.vue'
import TrackRow from './TrackRow.vue'
import LocaleToggle from './LocaleToggle.vue'
import { toast } from 'vue-sonner'
import { Toaster } from '@/components/ui/sonner'
import { t, tError } from './i18n'
import { fmtDuration } from './types'
import type { State, Track } from './types'

const SEARCH_DEBOUNCE_MS = 300

const name = ref('')
const nameInput = ref('')
const state = ref<State | null>(null)
const query = ref('')
const results = ref<Track[]>([])
const searching = ref(false)
const error = ref('')
const connected = ref(false)
const blocked = ref(false)

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
    if (e instanceof ApiError && e.code === 'blocked') { blocked.value = true; return }
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

const request = (tr: Track) => act(async () => {
  await api('POST', '/api/queue', tr)
  query.value = ''
  results.value = []
  return t('toast.requested', { title: tr.title })
})
const vote = (id: string) => act(async () => { await api('POST', `/api/queue/${id}/vote`); return t('toast.voted') })
const remove = (id: string) => act(async () => { await api('DELETE', `/api/queue/${id}`); return t('toast.removed') })
const voteSkip = () => act(async () => { await api('POST', '/api/skip'); return t('toast.skipVoted') })

const isSpotify = computed(() => state.value?.source?.id === 'spotify')

let es: EventSource | null = null
onMounted(async () => {
  await act(async () => {
    const me = await api<{ name: string }>('GET', '/api/me')
    name.value = me.name
  })
  if (blocked.value) return
  es = new EventSource('/api/events')
  es.addEventListener('state', (e) => {
    const s: State = JSON.parse((e as MessageEvent).data)
    state.value = s
    announce(s.event)
  })
  es.onopen = () => { connected.value = true }
  es.onerror = () => {
    connected.value = false
    // A kick just reconnects; a block shows up as 403 on the next probe.
    act(() => api('GET', '/api/me'))
  }
})
onUnmounted(() => es?.close())
</script>

<template>
  <!-- blocked -->
  <main v-if="blocked" class="min-h-screen flex items-center justify-center p-6">
    <Card class="w-full max-w-sm">
      <CardHeader>
        <p class="eyebrow">vibe-music</p>
        <CardTitle class="text-xl">{{ t('blocked.title') }}</CardTitle>
      </CardHeader>
      <CardContent class="text-sm text-body">{{ t('blocked.body') }}</CardContent>
    </Card>
  </main>

  <!-- name gate -->
  <main v-else-if="!name" class="min-h-screen flex items-center justify-center p-6">
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
  <main v-else class="min-h-screen mx-auto max-w-3xl space-y-4 p-4 sm:p-8">
    <Toaster position="bottom-right" rich-colors close-button />
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
          <div class="relative">
            <Search class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input v-model="query" class="pl-8" :placeholder="t('search.placeholder')" @input="onQuery" />
          </div>
        </CardHeader>
        <CardContent>
          <div v-if="searching" class="space-y-3">
            <div v-for="i in 3" :key="i" class="flex items-center gap-3">
              <Skeleton class="size-11 rounded-md" />
              <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
            </div>
          </div>
          <ScrollArea v-else-if="results.length" class="max-h-96">
            <div class="divide-y">
              <TrackRow v-for="tr in results" :key="tr.id" :track="tr" :subtitle="tr.album">
                <Button size="sm" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
              </TrackRow>
            </div>
          </ScrollArea>
          <p v-else class="text-sm text-muted-foreground">{{ query ? t('search.empty') : t('search.hint') }}</p>
        </CardContent>
      </Card>

      <!-- queue -->
      <Card>
        <CardHeader>
          <p class="eyebrow">{{ t('queue.eyebrow', { n: state?.queue.length ?? 0 }) }}</p>
        </CardHeader>
        <CardContent>
          <ScrollArea v-if="state?.queue.length" class="max-h-96">
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
          </ScrollArea>
          <p v-else class="text-sm text-muted-foreground">{{ t('queue.empty') }}</p>
        </CardContent>
      </Card>
    </div>

    <template v-if="isSpotify">
      <Separator />
      <footer class="pb-4 text-center text-xs text-muted-foreground">{{ t('spotify.footer') }}</footer>
    </template>
  </main>
</template>
