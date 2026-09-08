<script setup lang="ts">
// Shell: gate screens, header, the routed page, footer, and the two overlays
// (bottom now-playing bar off the home page, duplicate-request confirmation).
// All state and actions live in ./state so pages share one SSE stream.
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Users } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Toaster } from '@/components/ui/sonner'
import { goDeps, repoURL, uiDeps } from '../../ui/attributions'
import GitHubIcon from './GitHubIcon.vue'
import LocaleToggle from './LocaleToggle.vue'
import NowPlayingBar from './NowPlayingBar.vue'
import { t } from './i18n'
import {
  act, api, blocked, connected, error, inviteInvalid, inviteRequired, isSpotify, isYouTube, loading, name, nameInput,
  offline, pendingRequest, saveName, startStream, state, submit, teardown, version,
} from './state'

const route = useRoute()
const router = useRouter()
const home = computed(() => route.path === '/')
// Pages nest home → search → artist → album. Back retraces the guest's own
// steps; a page opened from a shared link has none, so it goes up one level.
const parent = computed(() => (route.path === '/search' ? '/' : '/search'))
function back() {
  if (window.history.state?.back) router.back()
  else router.replace(parent.value)
}
const showBar = computed(() => !home.value && !!state.value?.nowPlaying)

function confirmRequest() {
  const p = pendingRequest.value
  pendingRequest.value = null
  if (p) submit(p.track)
}

onMounted(async () => {
  // The invitation is redeemed from here, not by the link's GET: chat apps
  // fetch links for previews and would spend the single use before the person.
  await router.isReady()
  const code = router.currentRoute.value.query.invitationCode
  if (typeof code === 'string' && code) {
    try { await api('POST', '/api/join', { code }) } catch { inviteInvalid.value = true }
    router.replace('/')
  }
  await act(async () => {
    const me = await api<{ name: string; version: string }>('GET', '/api/me')
    name.value = me.name
    version.value = me.version
  })
  loading.value = false
  if (blocked.value || inviteRequired.value || offline.value) return
  startStream()
})
onUnmounted(teardown)
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
  <main v-else class="min-h-screen mx-auto max-w-3xl space-y-4 p-4 sm:p-8" :class="[offline ? 'pointer-events-none select-none opacity-50' : '', showBar ? 'pb-28' : '']" :aria-disabled="offline">
    <header class="flex items-center justify-between gap-2">
      <div class="flex min-w-0 items-center gap-2">
        <Button v-if="!home" variant="ghost" size="icon-sm" :aria-label="t('home.back')" :title="t('home.back')" @click="back"><ArrowLeft /></Button>
        <div class="min-w-0">
          <p class="eyebrow"><RouterLink to="/" class="hover:underline">vibe-music</RouterLink></p>
          <h1 class="truncate text-xl font-semibold tracking-tight">{{ t('header.hi', { name }) }}</h1>
        </div>
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

    <RouterView />

    <Separator />
    <footer class="space-y-1 pb-4 text-center text-xs text-muted-foreground">
      <p v-if="isSpotify || isYouTube">{{ t(isYouTube ? 'youtube.footer' : 'spotify.footer') }}</p>
      <Dialog>
        <DialogTrigger as-child><button type="button" class="underline-offset-4 hover:underline">{{ t('about.open') }}</button></DialogTrigger>
        <DialogContent class="max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{{ t('about.title') }}</DialogTitle>
            <DialogDescription class="break-keep">{{ t('about.desc') }}</DialogDescription>
            <p v-if="version" class="font-mono text-xs text-muted-foreground">vibe-music v{{ version }}</p>
          </DialogHeader>
          <Button variant="outline" class="w-full" as-child>
            <a :href="repoURL" target="_blank" rel="noopener"><GitHubIcon />{{ t('about.github') }}</a>
          </Button>
          <div v-for="[label, deps] in [['about.go', goDeps], ['about.ui', uiDeps]] as const" :key="label" class="space-y-2">
            <p class="eyebrow">{{ t(label) }}</p>
            <ul class="divide-y rounded-md border text-left">
              <li v-for="d in deps" :key="d.name" class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                <a :href="d.url" target="_blank" rel="noopener" class="truncate hover:underline">{{ d.name }}</a>
                <Badge variant="secondary" class="shrink-0 font-mono">{{ d.license }}</Badge>
              </li>
            </ul>
          </div>
        </DialogContent>
      </Dialog>
    </footer>

    <NowPlayingBar v-if="showBar" />

    <!-- requesting a song that is playing or already queued -->
    <Dialog :open="!!pendingRequest" @update:open="(o: boolean) => { if (!o) pendingRequest = null }">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ t('search.dupTitle') }}</DialogTitle>
          <DialogDescription class="break-keep">
            {{ pendingRequest?.playing ? t('search.dupPlaying', { title: pendingRequest.track.title }) : t('search.dupQueued', { title: pendingRequest?.track.title ?? '', n: pendingRequest?.position ?? 0 }) }}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" @click="pendingRequest = null">{{ t('search.cancel') }}</Button>
          <Button @click="confirmRequest">{{ t('search.dupConfirm') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </main>
</template>
