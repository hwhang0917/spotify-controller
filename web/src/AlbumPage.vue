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
import type { Album } from './types'

const props = defineProps<{ source: string; id: string }>()
const page = ref<Album | null>(null)
const loading = ref(true)
watch(() => [props.source, props.id], async () => {
  loading.value = true
  page.value = null
  await act(async () => { page.value = await api<Album>('GET', `/api/album?source=${encodeURIComponent(props.source)}&id=${encodeURIComponent(props.id)}`) })
  loading.value = false
}, { immediate: true })
</script>

<template>
  <Card>
    <CardHeader class="flex-row items-center gap-4">
      <Artwork :src="page?.artworkUrl" :alt="page?.name" class="size-32" />
      <div class="min-w-0 space-y-1">
        <p class="eyebrow flex items-center gap-1.5"><SourceIcon :source="source" class="size-3" />{{ t('browse.album') }}</p>
        <Skeleton v-if="loading" class="h-7 w-40" />
        <template v-else-if="page">
          <h1 class="truncate text-2xl font-semibold tracking-tight">{{ page.name }}</h1>
          <p class="truncate text-sm text-body">
            <RouterLink v-if="page.artistId" :to="`/artist/${source}/${page.artistId}`" class="hover:underline">{{ page.artist }}</RouterLink>
            <template v-else>{{ page.artist }}</template>
            <template v-if="page.year"> · {{ page.year }}</template>
          </p>
        </template>
        <h1 v-else class="text-2xl font-semibold tracking-tight">{{ t('browse.notFound') }}</h1>
      </div>
    </CardHeader>
    <CardContent v-if="loading" class="space-y-3">
      <div v-for="i in 5" :key="i" class="flex items-center gap-3">
        <Skeleton class="size-11 rounded-md" />
        <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
      </div>
    </CardContent>
    <CardContent v-else-if="page?.tracks?.length" class="divide-y">
      <TrackRow v-for="(tr, i) in page.tracks" :key="tr.id" :track="tr" :index="i + 1" hearts>
        <Button size="sm" :disabled="!isEnabled(tr.source)" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
      </TrackRow>
    </CardContent>
    <CardContent v-else class="text-sm text-muted-foreground">{{ t('browse.notFound') }}</CardContent>
  </Card>
</template>
