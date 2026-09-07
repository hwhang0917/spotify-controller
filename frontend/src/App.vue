<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ArrowDown, ArrowUp, Ban, Check, ChevronUp, Copy, FolderOpen, FolderPlus, Minus, Pause, Play, Power, RefreshCw, RotateCw, SkipForward, Ticket, Trash2, Unplug, Users, Volume2, X } from '@lucide/vue'
import * as api from '../wailsjs/go/main/App'
import { EventsOn, WindowReload } from '../wailsjs/runtime/runtime'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import Artwork from './Artwork.vue'
import SpotifyIcon from './SpotifyIcon.vue'
import YouTubePlayer from './YouTubePlayer.vue'
import YouTubeIcon from './YouTubeIcon.vue'
import HelpTip from './HelpTip.vue'
import { toast } from 'vue-sonner'
import { Toaster } from '@/components/ui/sonner'
import { locale, setLocale, t, tError } from './i18n'
import type { Config, Device, GuestInfo, Invitation, ServerStatus, SourceStatus, State } from './types'
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
const invitations = ref<Invitation[]>([])
const inviteTtl = ref('480')
const TTL_OPTIONS = [['60', 'invite.ttl.1h'], ['480', 'invite.ttl.8h'], ['1440', 'invite.ttl.24h'], ['10080', 'invite.ttl.7d']] as const
const devices = ref<Device[]>([])
const folders = ref<string[]>([])
const volume = ref([100])
const skipRatio = ref([50])
const busy = ref('')

// Every action reports through toasts: red for failures (translated when the
// code is known), green for the success message the caller returns.
async function run(label: string, fn: () => Promise<string | void>, errVars: Record<string, string | number> = {}) {
  busy.value = label
  try {
    const msg = await fn()
    if (msg) toast.success(msg)
  } catch (e) {
    toast.error(tError(e, errVars))
  } finally {
    busy.value = ''
    await refresh().catch((e) => toast.error(tError(e)))
  }
}

async function refresh() {
  const [c, s, st, ps, g, inv] = await Promise.all([api.GetConfig(), api.Sources(), api.Status(), api.GetState(), api.Guests(), api.Invitations()])
  cfg.value = c
  sources.value = s
  server.value = { ...st, port: server.value.port || st.port }
  state.value = ps
  guests.value = g ?? []
  invitations.value = inv ?? []
  folders.value = [...(c.local.folders ?? [])]
  skipRatio.value = [Math.round(c.skipRatio * 100)]
}

function configToSave() {
  return { ...cfg.value!, skipRatio: skipRatio.value[0] / 100, local: { folders: folders.value } }
}
const saveConfig = () => cfg.value && run('save', async () => { await api.SaveConfig(configToSave()); return t('toast.saved') })

// Native directory chooser; saving also rescans so the change is audible right away.
const addFolder = () => run('pick', async () => {
  const dir = await api.PickFolder()
  if (!dir || folders.value.includes(dir)) return
  folders.value = [...folders.value, dir]
  await api.SaveConfig(configToSave())
  return t('local.indexed', { n: await api.LocalRescan() })
})
const removeFolder = (dir: string) => run('remove-folder', async () => {
  folders.value = folders.value.filter((f) => f !== dir)
  await api.SaveConfig(configToSave())
  return t('local.indexed', { n: await api.LocalRescan() })
})

const toggleServer = () => run('server', async () => {
  if (server.value.running) { await api.StopServer(); return t('toast.serverStopped') }
  const url = await api.StartServer(server.value.port)
  return t('toast.serverStarted', { url })
}, { port: server.value.port })
const useSource = (id: string) => run(id, async () => {
  await api.SetActiveSource(id)
  return t('toast.sourceSwitched', { name: sources.value.find((s) => s.id === id)?.name ?? id })
})
const rescan = () => run('rescan', async () => t('local.indexed', { n: await api.LocalRescan() }))
const connect = () => run('connect', async () => {
  await api.SaveConfig(configToSave())
  await api.SpotifyConnect()
  return t('spotify.connected')
})
const disconnect = () => run('disconnect', async () => { await api.SpotifyDisconnect(); return t('toast.spotifyDisconnected') })
const loadDevices = () => run('devices', async () => {
  devices.value = await api.SpotifyDevices()
  return t('toast.devicesLoaded', { n: devices.value.length })
})
const togglePlay = () => run('play', () => (state.value?.nowPlaying?.playing ? api.Pause() : api.Resume()))
const skip = () => run('skip', async () => { await api.Skip(); return t('toast.skipped') })
const setVolume = (v: number[] | undefined) => v && run('volume', () => api.SetVolume(v[0]))
const removeItem = (id: string) => run('remove-item', async () => { await api.RemoveQueueItem(id); return t('toast.queueRemoved') })
const moveItem = (id: string, index: number) => run('move-item', async () => { await api.MoveQueueItem(id, index); return t('toast.queueMoved') })

// Seek slider: follows playback until the admin grabs it, then commits once.
const seeking = ref<number[] | null>(null)
const seekValue = computed(() => seeking.value ?? [positionMs.value])
const durationMs = computed(() => (np.value?.track.duration ?? 0) / NS_PER_MS)
const onSeekInput = (v: number[] | undefined) => { if (v) seeking.value = v }
const onSeekCommit = (v: number[] | undefined) => {
  seeking.value = null
  if (!v || !np.value) return
  run('seek', async () => { await api.Seek(Math.round(v[0])); return t('toast.seeked', { pos: fmtDuration(v[0] * NS_PER_MS) }) })
}
const kick = (id: string) => run('kick', async () => { await api.KickGuest(id); return t('toast.guestKicked') })
const removeGuest = (id: string) => run('remove', async () => { await api.RemoveGuest(id); return t('toast.guestRemoved') })
const youtubeKey = ref('')
const saveYouTubeKey = () => run('yt-key', async () => {
  await api.SetYouTubeAPIKey(youtubeKey.value)
  const had = youtubeKey.value.trim() !== ''
  youtubeKey.value = ''
  return t(had ? 'toast.youtubeKeySaved' : 'toast.youtubeKeyCleared')
})
const isYouTube = computed(() => state.value?.source?.id === 'youtube')

// Enable/disable a source. Switching off the one that is playing or has a
// queue asks first, because it stops playback and clears the queue.
const pendingDisable = ref<SourceStatus | null>(null)
const applyEnabled = (s: SourceStatus, on: boolean) => run('source-enabled', async () => {
  await api.SetSourceEnabled(s.id, on)
  return t(on ? 'toast.sourceEnabled' : 'toast.sourceDisabled', { name: s.name })
})
function toggleEnabled(s: SourceStatus, on: boolean) {
  const inUse = s.active && (state.value?.nowPlaying || (state.value?.queue.length ?? 0) > 0)
  if (!on && inUse) pendingDisable.value = s
  else applyEnabled(s, on)
}
function confirmDisable() {
  const s = pendingDisable.value
  pendingDisable.value = null
  if (s) applyEnabled(s, false)
}
const setInviteOnly = (on: boolean) => run('invite-only', async () => { await api.SetInviteOnly(on); return t(on ? 'toast.inviteOn' : 'toast.inviteOff') })
const admit = (id: string) => run('admit', async () => { await api.AdmitGuest(id); return t('toast.guestAdmitted') })
const createInvitation = () => run('invite', async () => {
  const inv: Invitation = await api.CreateInvitation(Number(inviteTtl.value))
  return t('toast.inviteCreated', { code: inv.code ?? '' })
})
const revokeInvitation = (id: number) => run('revoke', async () => { await api.RevokeInvitation(id); return t('toast.inviteRevoked') })
const joinUrl = (code: string) => `${server.value.url || `http://<host>:${server.value.port}`}/join?invitationCode=${code}`
const copyText = (text: string) => run('copy', async () => { await navigator.clipboard.writeText(text); return t('toast.linkCopied') })
const copyLink = (code: string) => copyText(joinUrl(code))
const isExpired = (inv: Invitation) => new Date(inv.expiresAt).getTime() < Date.now()
const fmtWhen = (iso: string) => new Date(iso).toLocaleString(locale.value === 'ko' ? 'ko-KR' : 'en-US', { dateStyle: 'short', timeStyle: 'short' })
const block = (g: GuestInfo) => run('block', async () => {
  await api.BlockGuest(g.id, !g.blocked)
  return t(g.blocked ? 'toast.guestUnblocked' : 'toast.guestBlocked')
})
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
const online = computed(() => guests.value.filter((g) => g.connections > 0).length)

// The webview has no browser chrome, so F5 / Ctrl+R (Cmd+R) reload by hand.
function onKey(e: KeyboardEvent) {
  if (e.key === 'F5' || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'r')) {
    e.preventDefault()
    WindowReload()
  }
}

const stops: Array<() => void> = []
let poll: number | undefined
let tick: number | undefined
onMounted(async () => {
  await refresh().catch((e) => toast.error(tError(e)))
  stops.push(EventsOn('state', (s: State) => { state.value = s }))
  stops.push(EventsOn('guests', (g: GuestInfo[]) => { guests.value = g ?? []; api.Invitations().then((i) => { invitations.value = i ?? [] }).catch(() => {}) }))
  window.addEventListener('keydown', onKey)
  tick = window.setInterval(() => { now.value = Date.now() }, 500)
  poll = window.setInterval(async () => {
    try {
      const [g, st] = await Promise.all([api.Guests(), api.Status()])
      guests.value = g ?? []
      server.value = { ...st, port: server.value.port || st.port }
    } catch {
      // transient; the next poll or event will catch up
    }
  }, POLL_MS)
})
onUnmounted(() => { stops.forEach((s) => s()); window.clearInterval(poll); window.clearInterval(tick); window.removeEventListener('keydown', onKey) })
</script>

<template>
  <TooltipProvider>
    <Toaster position="bottom-right" rich-colors close-button />
    <main class="mx-auto min-h-screen max-w-6xl space-y-6 p-6 lg:p-8">
      <header class="flex items-center justify-between">
        <div>
          <p class="eyebrow">{{ t('admin') }}</p>
          <h1 class="text-2xl font-semibold tracking-tight">vibe-music</h1>
        </div>
        <div class="flex items-center gap-3">
          <Badge v-if="server.running" variant="outline" class="gap-1.5">
            <span class="size-1.5 rounded-full bg-link" /><Users />{{ t('server.guests', { n: online }) }}
          </Badge>
          <Button :variant="server.running ? 'outline' : 'default'" :disabled="!!busy" @click="toggleServer">
            <Power />{{ server.running ? t('server.stop') : t('server.start') }}
          </Button>
          <Tooltip>
            <TooltipTrigger as-child>
              <Button variant="ghost" size="icon-sm" :aria-label="t('reload')" @click="WindowReload()"><RotateCw /></Button>
            </TooltipTrigger>
            <TooltipContent>{{ t('reload') }} · F5</TooltipContent>
          </Tooltip>
          <Button variant="ghost" size="xs" class="font-mono text-muted-foreground" @click="setLocale(locale === 'en' ? 'ko' : 'en')">
            {{ locale === 'en' ? 'KO' : 'EN' }}
          </Button>
        </div>
      </header>

      <div v-if="cfg" class="grid gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <!-- Player -->
        <Card class="overflow-hidden self-start">
          <CardContent class="p-0">
            <div class="flex flex-col" :class="isYouTube ? '' : 'sm:flex-row'">
              <YouTubePlayer v-if="isYouTube">
                <template #blocked>{{ t('youtube.blocked') }}</template>
              </YouTubePlayer>
              <Artwork v-else :src="np?.track.artworkUrl?.startsWith('http') ? np.track.artworkUrl : undefined" class="aspect-square w-full sm:w-64 rounded-none" />
              <div class="flex flex-1 flex-col justify-between gap-5 px-8 py-6">
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
                    <Slider
                      :model-value="seekValue" :max="Math.max(durationMs, 1)" :step="500" :disabled="!np || !durationMs"
                      class="cursor-pointer"
                      @update:model-value="onSeekInput" @value-commit="onSeekCommit"
                    />
                    <div class="flex justify-between font-mono text-xs text-muted-foreground">
                      <span>{{ fmtDuration(seekValue[0] * NS_PER_MS) }}</span>
                      <span>{{ fmtDuration(np?.track.duration ?? 0) }}</span>
                    </div>
                  </div>
                  <div class="flex items-center gap-3">
                    <Button size="icon-lg" :disabled="!np || !!busy" @click="togglePlay" :aria-label="np?.playing ? t('now.pause') : t('now.resume')">
                      <Pause v-if="np?.playing" /><Play v-else />
                    </Button>
                    <Button variant="outline" size="icon-lg" :disabled="!np || !!busy" @click="skip" :aria-label="t('now.skip')">
                      <SkipForward />
                    </Button>
                    <Volume2 class="ml-2 size-4 text-muted-foreground" />
                    <Slider v-model="volume" :max="100" :step="1" class="flex-1" @value-commit="setVolume" />
                    <span class="w-10 text-right font-mono text-xs text-muted-foreground tabular-nums">{{ volume[0] }}%</span>
                  </div>
                </div>
              </div>
            </div>

          </CardContent>
        </Card>

        <!-- Queue -->
        <Card class="lg:col-start-1 lg:row-start-2">
          <CardHeader>
            <CardTitle>{{ t('queue.title') }} <Badge variant="secondary" class="ml-1">{{ state?.queue.length ?? 0 }}</Badge></CardTitle>
            <CardDescription>{{ t('queue.desc') }}</CardDescription>
          </CardHeader>
          <CardContent class="p-0">
            <p v-if="!state?.queue.length" class="px-6 pb-6 text-sm text-muted-foreground">{{ t('queue.emptyAdmin') }}</p>
            <ScrollArea v-else class="max-h-[28rem]">
              <ul class="divide-y">
                <li v-for="(it, i) in state.queue" :key="it.id" class="flex items-center gap-3 px-6 py-2.5 text-sm">
                  <span class="w-5 text-right font-mono text-xs text-muted-foreground">{{ i + 1 }}</span>
                  <Artwork :src="it.track.artworkUrl?.startsWith('http') ? it.track.artworkUrl : undefined" class="size-10" />
                  <div class="min-w-0 flex-1">
                    <p class="truncate font-medium">{{ it.track.title }}</p>
                    <p class="truncate text-xs text-muted-foreground">{{ it.track.artist || '—' }} · {{ it.requestedBy }}</p>
                  </div>
                  <span class="font-mono text-xs text-muted-foreground tabular-nums">{{ fmtDuration(it.track.duration) }}</span>
                  <Badge variant="outline" class="font-mono tabular-nums"><ChevronUp />{{ it.votes }}</Badge>
                  <div class="flex items-center">
                    <Button variant="ghost" size="icon-sm" :disabled="i === 0 || !!busy" :aria-label="t('queue.up')" @click="moveItem(it.id, i - 1)"><ArrowUp /></Button>
                    <Button variant="ghost" size="icon-sm" :disabled="i === state.queue.length - 1 || !!busy" :aria-label="t('queue.down')" @click="moveItem(it.id, i + 1)"><ArrowDown /></Button>
                    <Tooltip>
                      <TooltipTrigger as-child>
                        <Button variant="ghost" size="icon-sm" class="text-destructive" :disabled="!!busy" @click="removeItem(it.id)"><X /></Button>
                      </TooltipTrigger>
                      <TooltipContent>{{ t('queue.remove') }}</TooltipContent>
                    </Tooltip>
                  </div>
                </li>
              </ul>
            </ScrollArea>
          </CardContent>
        </Card>

        <!-- Controls -->
        <Tabs default-value="server" class="min-w-0 lg:col-start-2 lg:row-start-1 lg:row-span-2">
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
                  <Input id="port" v-model.number="server.port" type="number" :disabled="server.running" class="w-32 font-mono" />
                  <p class="text-xs text-muted-foreground">{{ t('server.portHint') }}</p>
                </div>
                <div v-if="server.running" class="space-y-1">
                  <Label>{{ t('server.joinUrl') }}</Label>
                  <div class="flex gap-2">
                    <p class="min-w-0 flex-1 select-all truncate rounded-md border bg-muted px-3 py-2 font-mono text-sm">{{ server.url }}</p>
                    <Button variant="outline" :disabled="!!busy" :aria-label="t('server.copyUrl')" @click="copyText(server.url)"><Copy />{{ t('server.copyUrl') }}</Button>
                  </div>
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

            <Card class="mt-4">
              <CardHeader>
                <CardTitle class="flex items-center justify-between gap-3">
                  <span class="flex items-center gap-2"><Ticket class="size-4" />{{ t('invite.title') }}</span>
                  <label class="flex items-center gap-2 text-sm font-normal">
                    {{ t('invite.only') }}
                    <Switch :model-value="cfg.inviteOnly" :disabled="!!busy" @update:model-value="setInviteOnly" />
                  </label>
                </CardTitle>
                <CardDescription>{{ t('invite.desc', { url: '/join?invitationCode=…' }) }}</CardDescription>
              </CardHeader>
              <CardContent class="space-y-4">
                <div class="flex flex-wrap items-end gap-2">
                  <div class="space-y-1">
                    <Label>{{ t('invite.ttl') }}</Label>
                    <Select v-model="inviteTtl">
                      <SelectTrigger class="w-36"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem v-for="[v, k] in TTL_OPTIONS" :key="v" :value="v">{{ t(k) }}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <Button :disabled="!!busy" @click="createInvitation"><Ticket />{{ t('invite.create') }}</Button>
                </div>
                <p v-if="!invitations.length" class="text-sm text-muted-foreground">{{ t('invite.empty') }}</p>
                <ul v-else class="divide-y rounded-md border">
                  <li v-for="inv in invitations" :key="inv.id" class="flex items-center gap-3 px-3 py-2 text-sm" :class="inv.revoked || isExpired(inv) ? 'opacity-60' : ''">
                    <div class="min-w-0 flex-1">
                      <p class="font-mono">
                        <template v-if="inv.code">{{ inv.code }}</template>
                        <span v-else class="text-muted-foreground">{{ t('invite.hidden', { label: inv.label }) }}</span>
                      </p>
                      <p class="text-xs text-muted-foreground">
                        <template v-if="inv.revoked">{{ t('invite.revoked') }}</template>
                        <template v-else-if="isExpired(inv)">{{ t('invite.expired') }}</template>
                        <template v-else>{{ t('invite.expires', { when: fmtWhen(inv.expiresAt) }) }}</template>
                        · {{ t('invite.uses', { n: inv.uses }) }}
                      </p>
                    </div>
                    <Button v-if="inv.code && !inv.revoked && !isExpired(inv)" variant="outline" size="sm" :disabled="!!busy" @click="copyLink(inv.code)"><Copy />{{ t('invite.copy') }}</Button>
                    <Button v-if="!inv.revoked && !isExpired(inv)" variant="ghost" size="sm" class="text-destructive" :disabled="!!busy" @click="revokeInvitation(inv.id)">{{ t('invite.revoke') }}</Button>
                  </li>
                </ul>
              </CardContent>
            </Card>
          </TabsContent>

          <!-- Sources -->
          <TabsContent value="sources" class="space-y-4">
            <div class="grid gap-3 sm:grid-cols-3">
              <div
                v-for="s in sources" :key="s.id"
                class="rounded-lg border bg-card p-4 transition-colors"
                :class="[s.active ? 'border-primary' : '', s.enabled ? '' : 'opacity-60']"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="flex min-w-0 items-center gap-2 font-medium">
                    <SpotifyIcon v-if="s.id === 'spotify'" /><YouTubeIcon v-else-if="s.id === 'youtube'" /><FolderOpen v-else class="size-4 text-muted-foreground" />
                    <span class="truncate">{{ s.name }}</span>
                  </span>
                  <label class="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
                    {{ t('source.use') }}
                    <Switch size="sm" :model-value="s.enabled" :disabled="!!busy" @update:model-value="(on: boolean) => toggleEnabled(s, on)" />
                  </label>
                </div>
                <div class="mt-3 flex items-center justify-between gap-2">
                  <Badge :variant="s.active ? 'default' : s.enabled && s.ready ? 'secondary' : 'outline'">
                    {{ !s.enabled ? t('source.disabled') : s.active ? t('source.active') : s.ready ? t('source.ready') : t('source.setup') }}
                  </Badge>
                  <Button v-if="!s.active" size="sm" variant="outline" :disabled="!s.enabled || !s.ready || !!busy" @click="useSource(s.id)">{{ t('source.useNow') }}</Button>
                </div>
                <p class="mt-2 truncate text-xs text-muted-foreground">{{ s.detail }}</p>
              </div>
            </div>

            <AlertDialog :open="!!pendingDisable" @update:open="(o: boolean) => { if (!o) pendingDisable = null }">
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{{ t('source.disableTitle', { name: pendingDisable?.name ?? '' }) }}</AlertDialogTitle>
                  <AlertDialogDescription>{{ t('source.disableBody') }}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{{ t('source.cancel') }}</AlertDialogCancel>
                  <AlertDialogAction @click="confirmDisable">{{ t('source.disableConfirm') }}</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>

            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2"><FolderOpen class="size-4" />{{ t('local.title') }}<HelpTip :text="t('local.help')" /></CardTitle>
              </CardHeader>
              <CardContent class="space-y-3">
                <ul v-if="folders.length" class="divide-y rounded-md border">
                  <li v-for="dir in folders" :key="dir" class="flex items-center gap-2 px-3 py-1.5">
                    <FolderOpen class="size-4 shrink-0 text-muted-foreground" />
                    <span class="min-w-0 flex-1 truncate font-mono text-xs" :title="dir">{{ dir }}</span>
                    <Button variant="ghost" size="icon-xs" :disabled="!!busy" :aria-label="t('local.remove')" @click="removeFolder(dir)"><Minus /></Button>
                  </li>
                </ul>
                <p v-else class="text-sm text-muted-foreground">{{ t('local.empty') }}</p>
                <div class="flex gap-2">
                  <Button :disabled="!!busy" @click="addFolder"><FolderPlus />{{ t('local.add') }}</Button>
                  <Button variant="outline" :disabled="!folders.length || !!busy" @click="rescan"><RefreshCw />{{ t('local.rescan') }}</Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2"><SpotifyIcon />{{ t('spotify.title') }}<HelpTip :text="t('spotify.help')" /></CardTitle>
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
                  <Button v-else variant="outline" :disabled="!!busy" @click="disconnect"><Unplug />{{ t('spotify.disconnect') }}</Button>
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
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle class="flex items-center gap-2"><YouTubeIcon />{{ t('youtube.title') }}<HelpTip :text="t('youtube.help')" /></CardTitle>
              </CardHeader>
              <CardContent class="space-y-3">
                <div class="space-y-2">
                  <Label for="ytKey">{{ t('youtube.apiKey') }}</Label>
                  <div class="flex gap-2">
                    <Input id="ytKey" v-model="youtubeKey" type="password" class="font-mono text-xs" :placeholder="cfg.youtube.hasKey ? '••••••••' : 'AIza…'" />
                    <Button :disabled="!!busy || (!youtubeKey.trim() && !cfg.youtube.hasKey)" @click="saveYouTubeKey">
                      {{ youtubeKey.trim() || !cfg.youtube.hasKey ? t('youtube.save') : t('youtube.clear') }}
                    </Button>
                  </div>
                  <p v-if="cfg.youtube.hasKey" class="text-xs text-muted-foreground"><Check class="mr-1 inline size-3" />{{ t('youtube.apiKeySet') }}</p>
                </div>
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
                        <div class="flex flex-wrap gap-1">
                          <Badge v-if="g.blocked" variant="destructive">{{ t('guests.blocked') }}</Badge>
                          <Badge v-else-if="g.connections" variant="secondary" class="gap-1.5"><span class="size-1.5 rounded-full bg-link" />{{ t('guests.online') }}</Badge>
                          <Badge v-else variant="outline">{{ t('guests.offline') }}</Badge>
                          <Badge v-if="cfg.inviteOnly && !g.blocked" :variant="g.admitted ? 'outline' : 'destructive'">{{ g.admitted ? t('guests.admitted') : t('guests.waiting') }}</Badge>
                        </div>
                      </TableCell>
                      <TableCell class="text-right">
                        <div class="inline-flex gap-1">
                          <Button v-if="cfg.inviteOnly && !g.admitted && !g.blocked" size="sm" :disabled="!!busy" @click="admit(g.id)"><Check />{{ t('guests.admit') }}</Button>
                          <Tooltip v-if="g.connections">
                            <TooltipTrigger as-child>
                              <Button variant="ghost" size="icon-sm" :disabled="!!busy" @click="kick(g.id)"><Unplug /></Button>
                            </TooltipTrigger>
                            <TooltipContent>{{ t('guests.kick') }} · {{ t('guests.kickHint') }}</TooltipContent>
                          </Tooltip>
                          <Tooltip v-if="!g.blocked">
                            <TooltipTrigger as-child>
                              <Button variant="ghost" size="icon-sm" :disabled="!!busy" @click="removeGuest(g.id)"><Trash2 /></Button>
                            </TooltipTrigger>
                            <TooltipContent>{{ t('guests.remove') }} · {{ t('guests.removeHint') }}</TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger as-child>
                              <Button :variant="g.blocked ? 'outline' : 'destructive'" size="sm" :disabled="!!busy" @click="block(g)">
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
