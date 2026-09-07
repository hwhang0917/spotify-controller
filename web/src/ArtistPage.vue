<script setup lang="ts">
import { ref, watch } from 'vue'
import { Plus } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import Artwork from './Artwork.vue'
import SourceIcon from './SourceIcon.vue'
import TrackRow from './TrackRow.vue'
import { t } from './i18n'
import { act, api, isEnabled, request } from './state'
import type { Artist } from './types'

const props = defineProps<{ source: string; id: string }>()
const page = ref<Artist | null>(null)
const loading = ref(true)
watch(() => [props.source, props.id], async () => {
  loading.value = true
  page.value = null
  await act(async () => { page.value = await api<Artist>('GET', `/api/artist?source=${encodeURIComponent(props.source)}&id=${encodeURIComponent(props.id)}`) })
  loading.value = false
}, { immediate: true })
</script>

<template>
  <Card>
    <CardHeader class="flex-row items-center gap-4">
      <Artwork :src="page?.artworkUrl" :alt="page?.name" class="size-24 rounded-full" />
      <div class="min-w-0">
        <p class="eyebrow flex items-center gap-1.5"><SourceIcon :source="source" class="size-3" />{{ t('browse.artist') }}</p>
        <Skeleton v-if="loading" class="mt-1 h-7 w-40" />
        <h1 v-else class="truncate text-2xl font-semibold tracking-tight">{{ page?.name ?? t('browse.notFound') }}</h1>
      </div>
    </CardHeader>
    <CardContent v-if="loading" class="space-y-3">
      <div v-for="i in 5" :key="i" class="flex items-center gap-3">
        <Skeleton class="size-11 rounded-md" />
        <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
      </div>
    </CardContent>
    <CardContent v-else-if="page" class="space-y-6">
      <section>
        <p class="eyebrow mb-1">{{ t('browse.tracks') }}</p>
        <div class="divide-y">
          <TrackRow v-for="(tr, i) in page.tracks" :key="tr.id" :track="tr" :index="i + 1" hearts>
            <Button size="sm" :disabled="!isEnabled(tr.source)" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
          </TrackRow>
        </div>
      </section>
      <section v-if="page.albums.length">
        <p class="eyebrow mb-2">{{ t('browse.albums') }}</p>
        <div class="grid grid-cols-3 gap-3 sm:grid-cols-5">
          <RouterLink v-for="al in page.albums" :key="al.id" :to="`/album/${source}/${al.id}`" class="group min-w-0 space-y-1">
            <Artwork :src="al.artworkUrl" :alt="al.name" class="aspect-square w-full" />
            <p class="truncate text-sm font-medium group-hover:underline">{{ al.name }}</p>
            <p v-if="al.year" class="text-xs text-muted-foreground">{{ al.year }}</p>
          </RouterLink>
        </div>
      </section>
    </CardContent>
    <CardContent v-else class="text-sm text-muted-foreground">{{ t('browse.notFound') }}</CardContent>
  </Card>
</template>
