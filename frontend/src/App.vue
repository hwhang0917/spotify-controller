<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Ban, ChevronUp, FolderOpen, Pause, Play, Power, RefreshCw, SkipForward, Trash2, Unplug, Users, Volume2 } from '@lucide/vue'
import * as api from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import Artwork from './Artwork.vue'
import { locale, setLocale, t } from './i18n'
import type { Config, Device, GuestInfo, ServerStatus, SourceStatus, State } from './types'
import { fmtDuration, NS_PER_SEC } from './types'

// Wails events push changes; the poll is a safety net so the guest list and
// server status never go stale if a push is missed.
const POLL_MS = 3000
const NS_PER_MS = 1e6

const cfg = ref<Config | null>(null)
const sources = ref<SourceStatus[]>([])
const server = ref<ServerStatus>({ running: false, port: 0, url: '' })
const state = ref<State | null>(null)
const guests = ref<GuestInfo[]>([])
const devices = ref<Device[]>([])
const foldersText = ref('')
const volume = ref([100])
const skipRatio = ref([50])
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
  server.value = { ...st, port: server.value.port || st.port }
  state.value = ps
  guests.value = g
  foldersText.value = c.local.folders.join('\n')
  skipRatio.value = [Math.round(c.skipRatio * 100)]
}

function saveConfig() {
  if (!cfg.value) return
  const c = {
    ...cfg.value,
    skipRatio: skipRatio.value[0] / 100,
    local: { folders: foldersText.value.split('\n').map((l) => l.trim()).filter(Boolean) },
  }
  return run('save', () => api.SaveConfig(c))
}

const toggleServer = () => run('server', () => (server.value.running ? api.StopServer() : api.StartServer(server.value.port)))
const useSource = (id: string) => run(id, () => api.SetActiveSource(id))
const rescan = () => run('rescan', async () => { notice.value = t('local.indexed', { n: await api.LocalRescan() }) })
const connect = () => run('connect', async () => { await saveConfig(); await api.SpotifyConnect(); notice.value = t('spotify.connected') })
const loadDevices = () => run('devices', async () => { devices.value = await api.SpotifyDevices() })
const togglePlay = () => run('play', () => (state.value?.nowPlaying?.playing ? api.Pause() : api.Resume()))
const setVolume = (v: number[] | undefined) => v && run('volume', () => api.SetVolume(v[0]))
const isReady = (id: string) => sources.value.find((s) => s.id === id)?.ready ?? false

// keep the volume slider in sync with the player unless the admin is dragging it
watch(() => state.value?.volume, (v) => { if (v !== undefined && busy.value !== 'volume') volume.value = [v] })

const now = ref(Date.now())
const np = computed(() => state.value?.nowPlaying ?? null)
const positionMs = computed(() => {
  if (!np.value) return 0
  const base = np.value.position / NS_PER_MS
  return np.value.playing ? base + (now.value - new Date(np.value.at).getTime()) : base
})
const pct = computed(() => {
  const d = np.value?.track.duration
  return d ? Math.min(100, (positionMs.value * NS_PER_MS / d) * 100) : 0
})
const online = computed(() => guests.value.filter((g) => g.connections > 0).length)

const stops: Array<() => void> = []
let poll: number | undefined
let tick: number | undefined
onMounted(async () => {
  try { await refresh() } catch (e) { error.value = String(e) }
  stops.push(EventsOn('state', (s: State) => { state.value = s }))
  stops.push(EventsOn('guests', (g: GuestInfo[]) => { guests.value = g }))
  tick = window.setInterval(() => { now.value = Date.now() }, 500)
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
onUnmounted(() => { stops.forEach((s) => s()); window.clearInterval(poll); window.clearInterval(tick) })
</script>

<template>
  <TooltipProvider>
    <main class="mx-auto min-h-screen max-w-6xl space-y-6 p-6 lg:p-8">
      <header class="flex items-center justify-between">
        <div>
          <p class="eyebrow">{{ t('admin') }}</p>
          <h1 class="text-2xl font-semibold tracking-tight">vibe-music</h1>
        </div>
        <div class="flex items-center gap-3">
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <p v-else-if="notice" class="text-sm text-link">{{ notice }}</p>
          <Badge v-if="server.running" variant="outline" class="gap-1.5">
            <span class="size-1.5 rounded-full bg-link" /><Users />{{ t('server.guests', { n: online }) }}
          </Badge>
          <Button variant="ghost" size="xs" class="font-mono text-muted-foreground" @click="setLocale(locale === 'en' ? 'ko' : 'en')">
            {{ locale === 'en' ? 'KO' : 'EN' }}
          </Button>
        </div>
      </header>

      <div v-if="cfg" class="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <!-- Player -->
        <Card class="overflow-hidden self-start">
          <CardContent class="p-0">
            <div class="flex flex-col sm:flex-row">
              <Artwork :src="np?.track.artworkUrl?.startsWith('http') ? np.track.artworkUrl : undefined" class="aspect-square w-full sm:w-64 rounded-none" />
              <div class="flex flex-1 flex-col justify-between gap-5 p-6">
                <div class="space-y-1">
                  <p class="eyebrow">{{ t('now') }}<span v-if="state?.source"> · {{ state.source.name }}</span></p>
                  <template v-if="np">
                    <h2 class="text-2xl font-semibold tracking-tight leading-tight line-clamp-2">{{ np.track.title }}</h2>
                    <p class="text-base text-body truncate">{{ np.track.artist || '—' }}</p>
                    <p v-if="np.track.album" class="text-sm text-muted-foreground truncate">{{ np.track.album }}</p>
                    <div class="flex flex-wrap gap-2 pt-1">
                      <Badge v-if="np.requestedBy" variant="secondary">{{ t('now.requestedBy', { name: np.requestedBy }) }}</Badge>
                      <Badge variant="outline">{{ t('now.skipVotes', { v: state?.skipVotes ?? 0, t: state?.skipThreshold ?? 1 }) }}</Badge>
                    </div>
                  </template>
                  <p v-else class="text-lg font-medium text-muted-foreground">{{ t('now.empty') }}</p>
                </div>

                <div class="space-y-4">
                  <div class="space-y-1.5">
                    <Progress :model-value="pct" class="h-1.5" />
                    <div class="flex justify-between font-mono text-xs text-muted-foreground">
                      <span>{{ fmtDuration(positionMs * NS_PER_MS) }}</span>
                      <span>{{ fmtDuration(np?.track.duration ?? 0) }}</span>
                    </div>
                  </div>
                  <div class="flex items-center gap-3">
                    <Button size="icon-lg" :disabled="!np || !!busy" @click="togglePlay" :aria-label="np?.playing ? t('now.pause') : t('now.resume')">
                      <Pause v-if="np?.playing" /><Play v-else />
                    </Button>
                    <Button variant="outline" size="icon-lg" :disabled="!np || !!busy" @click="run('skip', api.Skip)" :aria-label="t('now.skip')">
                      <SkipForward />
                    </Button>
                    <Volume2 class="ml-2 size-4 text-muted-foreground" />
                    <Slider v-model="volume" :max="100" :step="1" class="flex-1" @value-commit="setVolume" />
                    <span class="w-10 text-right font-mono text-xs text-muted-foreground tabular-nums">{{ volume[0] }}%</span>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="state?.queue.length" class="border-t">
              <p class="eyebrow px-6 pt-4">{{ t('queue', { n: state.queue.length }) }}</p>
              <ScrollArea class="max-h-72">
                <div class="divide-y px-6 pb-2">
                  <div v-for="(it, i) in state.queue" :key="it.id" class="flex items-center gap-3 py-2 text-sm">
                    <span class="w-5 text-right font-mono text-xs text-muted-foreground">{{ i + 1 }}</span>
                    <Artwork :src="it.track.artworkUrl?.startsWith('http') ? it.track.artworkUrl : undefined" class="size-9" />
                    <div class="min-w-0 flex-1">
                      <p class="truncate font-medium">{{ it.track.title }}</p>
                      <p class="truncate text-xs text-muted-foreground">{{ it.track.artist || '—' }} · {{ it.requestedBy }}</p>
                    </div>
                    <Badge variant="outline" class="font-mono tabular-nums"><ChevronUp />{{ it.votes }}</Badge>
                  </div>
                </div>
              </ScrollArea>
            </div>
          </CardContent>
        </Card>

        <!-- Controls -->
        <Tabs default-value="server" class="min-w-0">
          <TabsList class="w-full">
            <TabsTrigger value="server" class="flex-1">{{ t('tab.server') }}</TabsTrigger>
            <TabsTrigger value="sources" class="flex-1">{{ t('tab.sources') }}</TabsTrigger>
            <TabsTrigger value="guests" class="flex-1">{{ t('tab.guests') }} <Badge variant="secondary" class="ml-1">{{ guests.length }}</Badge></TabsTrigger>
          </TabsList>

          <!-- Server -->
          <TabsContent value="server">
            <Card>
              <CardHeader>
                <CardTitle>{{ t('server.title') }}</CardTitle>
                <CardDescription>{{ t('server.desc') }}</CardDescription>
              </CardHeader>
              <CardContent class="space-y-5">
                <div class="space-y-2">
                  <Label for="port">{{ t('server.port') }}</Label>
                  <div class="flex gap-2">
                    <Input id="port" v-model.number="server.port" type="number" :disabled="server.running" class="w-32 font-mono" />
                    <Button :variant="server.running ? 'outline' : 'default'" :disabled="!!busy" @click="toggleServer">
                      <Power />{{ server.running ? t('server.stop') : t('server.start') }}
                    </Button>
                  </div>
                </div>
                <div v-if="server.running" class="space-y-1">
                  <Label>{{ t('server.joinUrl') }}</Label>
                  <p class="select-all rounded-md border bg-muted px-3 py-2 font-mono text-sm">{{ server.url }}</p>
                </div>
                <div class="space-y-2">
                  <div class="flex justify-between">
                    <Label>{{ t('server.skipRatio') }}</Label>
                    <span class="font-mono text-sm tabular-nums">{{ skipRatio[0] }}%</span>
                  </div>
                  <Slider v-model="skipRatio" :min="10" :max="100" :step="10" @value-commit="saveConfig" />
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <!-- Sources -->
          <TabsContent value="sources" class="space-y-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <button
                v-for="s in sources" :key="s.id"
                :disabled="s.active || !s.ready || !!busy"
                class="rounded-lg border bg-card p-4 text-left transition-colors hover:bg-muted disabled:cursor-default disabled:hover:bg-card"
                :class="s.active ? 'border-primary' : ''"
                @click="useSource(s.id)"
              >
                <div class="flex items-center justify-between">
                  <span class="font-medium">{{ s.name }}</span>
                  <Badge :variant="s.active ? 'default' : s.ready ? 'secondary' : 'outline'">
                    {{ s.active ? t('source.active') : s.ready ? t('source.ready') : t('source.setup') }}
                  </Badge>
                </div>
                <p class="mt-1 text-sm text-muted-foreground">{{ s.detail }}</p>
              </button>
            </div>

            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2"><FolderOpen class="size-4" />{{ t('local.title') }}</CardTitle>
                <CardDescription>{{ t('local.desc') }}</CardDescription>
              </CardHeader>
              <CardContent class="space-y-3">
                <Textarea v-model="foldersText" rows="3" class="font-mono text-xs" placeholder="C:\Users\me\Music" />
                <div class="flex gap-2">
                  <Button :disabled="!!busy" @click="saveConfig">{{ t('local.save') }}</Button>
                  <Button variant="outline" :disabled="!!busy" @click="rescan"><RefreshCw />{{ t('local.rescan') }}</Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{{ t('spotify.title') }}</CardTitle>
                <CardDescription>{{ t('spotify.desc') }}</CardDescription>
              </CardHeader>
              <CardContent class="space-y-4">
                <div class="space-y-2">
                  <Label for="clientId">{{ t('spotify.clientId') }}</Label>
                  <Input id="clientId" v-model="cfg.spotify.clientId" class="font-mono text-xs" />
                </div>
                <div class="flex flex-wrap gap-2">
                  <Button v-if="!isReady('spotify')" :disabled="!cfg.spotify.clientId || !!busy" @click="connect">
                    {{ busy === 'connect' ? t('spotify.connecting') : t('spotify.connect') }}
                  </Button>
                  <Button v-else variant="outline" :disabled="!!busy" @click="run('disconnect', api.SpotifyDisconnect)"><Unplug />{{ t('spotify.disconnect') }}</Button>
                  <Button variant="outline" :disabled="!isReady('spotify') || !!busy" @click="loadDevices">{{ t('spotify.devices') }}</Button>
                </div>
                <div v-if="devices.length" class="space-y-2">
                  <Label>{{ t('spotify.device') }}</Label>
                  <Select v-model="cfg.spotify.deviceId" @update:model-value="saveConfig">
                    <SelectTrigger class="w-full"><SelectValue :placeholder="t('spotify.deviceAny')" /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="">{{ t('spotify.deviceAny') }}</SelectItem>
                      <SelectItem v-for="d in devices" :key="d.id" :value="d.id">
                        {{ d.name }} · {{ d.type }}{{ d.active ? ' · ' + t('spotify.deviceActive') : '' }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <p class="text-xs text-muted-foreground">
                  {{ t('spotify.redirect') }} <span class="select-all font-mono text-foreground">http://127.0.0.1/callback</span>
                </p>
              </CardContent>
            </Card>
          </TabsContent>

          <!-- Guests -->
          <TabsContent value="guests">
            <Card>
              <CardHeader>
                <CardTitle>{{ t('guests.title') }}</CardTitle>
                <CardDescription>{{ t('guests.desc') }}</CardDescription>
              </CardHeader>
              <CardContent class="p-0">
                <p v-if="!guests.length" class="px-6 pb-6 text-sm text-muted-foreground">{{ t('guests.empty') }}</p>
                <Table v-else>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{{ t('guests.name') }}</TableHead>
                      <TableHead>{{ t('guests.status') }}</TableHead>
                      <TableHead class="text-right">{{ t('guests.actions') }}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow v-for="g in guests" :key="g.id">
                      <TableCell>
                        <span :class="g.name ? 'font-medium' : 'text-muted-foreground'">{{ g.name || t('guests.unnamed') }}</span>
                        <span class="ml-2 font-mono text-xs text-muted-foreground">{{ g.id.slice(0, 8) }}</span>
                      </TableCell>
                      <TableCell>
                        <Badge v-if="g.blocked" variant="destructive">{{ t('guests.blocked') }}</Badge>
                        <Badge v-else-if="g.connections" variant="secondary" class="gap-1.5"><span class="size-1.5 rounded-full bg-link" />{{ t('guests.online') }}</Badge>
                        <Badge v-else variant="outline">{{ t('guests.offline') }}</Badge>
                      </TableCell>
                      <TableCell class="text-right">
                        <div class="inline-flex gap-1">
                          <Tooltip v-if="g.connections">
                            <TooltipTrigger as-child>
                              <Button variant="ghost" size="icon-sm" :disabled="!!busy" @click="run('kick', () => api.KickGuest(g.id))"><Unplug /></Button>
                            </TooltipTrigger>
                            <TooltipContent>{{ t('guests.kick') }} · {{ t('guests.kickHint') }}</TooltipContent>
                          </Tooltip>
                          <Tooltip v-if="!g.blocked">
                            <TooltipTrigger as-child>
                              <Button variant="ghost" size="icon-sm" :disabled="!!busy" @click="run('remove', () => api.RemoveGuest(g.id))"><Trash2 /></Button>
                            </TooltipTrigger>
                            <TooltipContent>{{ t('guests.remove') }} · {{ t('guests.removeHint') }}</TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger as-child>
                              <Button :variant="g.blocked ? 'outline' : 'destructive'" size="sm" :disabled="!!busy" @click="run('block', () => api.BlockGuest(g.id, !g.blocked))">
                                <Ban />{{ g.blocked ? t('guests.unblock') : t('guests.block') }}
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>{{ t('guests.blockHint') }}</TooltipContent>
                          </Tooltip>
                        </div>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </main>
  </TooltipProvider>
</template>
