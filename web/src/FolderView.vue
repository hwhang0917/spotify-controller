<script setup lang="ts">
// One level of a source's folder tree: breadcrumb, subfolders, then the
// songs in this folder. Used inline on the Folders tab and by FolderPage.
import { ref, watch } from 'vue'
import { ChevronRight, Folder as FolderIcon, Plus } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import TrackRow from './TrackRow.vue'
import { t } from './i18n'
import { act, api, isEnabled, request } from './state'
import type { Folder } from './types'

const props = defineProps<{ source: string; id: string }>()
const page = ref<Folder | null>(null)
const loading = ref(true)
const link = (id: string) => (id ? `/folder/${props.source}/${id}` : `/folder/${props.source}`)
watch(() => [props.source, props.id], async () => {
  loading.value = true
  page.value = null
  await act(async () => { page.value = await api<Folder>('GET', `/api/folder?source=${encodeURIComponent(props.source)}&id=${encodeURIComponent(props.id)}`) })
  loading.value = false
}, { immediate: true })
</script>

<template>
  <div v-if="loading" class="space-y-3">
    <div v-for="i in 4" :key="i" class="flex items-center gap-3">
      <Skeleton class="size-9 rounded-md" />
      <Skeleton class="h-3 w-1/2" />
    </div>
  </div>
  <div v-else-if="page" class="space-y-3">
    <nav v-if="page.path?.length" class="flex flex-wrap items-center gap-1 text-xs text-muted-foreground" aria-label="breadcrumb">
      <template v-for="crumb in page.path" :key="crumb.id">
        <RouterLink :to="link(crumb.id)" class="hover:underline">{{ crumb.name || t('browse.allFolders') }}</RouterLink>
        <ChevronRight class="size-3" />
      </template>
      <span class="font-medium text-foreground">{{ page.name }}</span>
    </nav>
    <div v-if="page.folders?.length" class="divide-y">
      <RouterLink v-for="d in page.folders" :key="d.id" :to="link(d.id)" class="flex items-center gap-3 py-2 hover:bg-muted/50">
        <FolderIcon class="size-5 shrink-0 text-muted-foreground" />
        <span class="min-w-0 flex-1 truncate text-sm font-medium">{{ d.name }}</span>
        <span class="text-xs text-muted-foreground">{{ t('browse.songs', { n: d.count }) }}</span>
        <ChevronRight class="size-4 text-muted-foreground" />
      </RouterLink>
    </div>
    <div v-if="page.tracks?.length" class="divide-y">
      <TrackRow v-for="(tr, i) in page.tracks" :key="tr.id" :track="tr" :index="i + 1" hearts>
        <Button size="sm" :disabled="!isEnabled(tr.source)" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
      </TrackRow>
    </div>
    <p v-if="!page.folders?.length && !page.tracks?.length" class="text-sm text-muted-foreground">{{ t('browse.notFound') }}</p>
  </div>
  <p v-else class="text-sm text-muted-foreground">{{ t('browse.notFound') }}</p>
</template>
