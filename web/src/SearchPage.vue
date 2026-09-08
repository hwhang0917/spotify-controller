<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Plus, Search, X } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import FolderView from './FolderView.vue'
import TrackRow from './TrackRow.vue'
import SourceIcon from './SourceIcon.vue'
import { favorites } from './favorites'
import { t } from './i18n'
import { browse, canSearch, chart, chartLoading, hasChart, hasFolders, isEnabled, onQuery, pick, query, region, request, results, search, searched, searching, selected, sources, top } from './state'
import type { Track } from './types'

const route = useRoute()
const router = useRouter()

// The submitted query lives in the URL so a reload or a shared link restores
// the results. The box itself may differ while the guest is still typing.
watch(searched, (q) => { if (q !== (route.query.q ?? '')) router.replace({ query: q ? { q } : {} }) })
onMounted(() => {
  const q = route.query.q
  if (typeof q === 'string' && q) { if (q !== searched.value) { query.value = q; search() } }
  else if (searched.value) router.replace({ query: { q: searched.value } })
})
function clear() {
  query.value = ''
  search()
}

const tabs = computed(() => [
  ...(hasChart.value ? [{ id: 'chart' as const, label: t('search.chart', { n: chart.value.length || 50, region: region.value }) }] : []),
  { id: 'top' as const, label: t('search.top') },
  { id: 'fav' as const, label: t('search.favorites') },
  ...(hasFolders.value ? [{ id: 'dir' as const, label: t('search.folders') }] : []),
])
const list = computed<Track[]>(() => browse.value === 'chart' ? chart.value : browse.value === 'fav' ? favorites.value : top.value)
const emptyText = computed(() => browse.value === 'chart' ? t('search.chartEmpty') : browse.value === 'fav' ? t('search.favEmpty') : t('search.hint'))
</script>

<template>
  <Card>
    <CardHeader class="space-y-3">
      <p class="eyebrow">{{ t('search.eyebrow') }}</p>
      <form class="flex gap-2" @submit.prevent="search">
        <div class="relative flex-1">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="query" type="search" enterkeyhint="search" class="pl-8 pr-8" :disabled="!canSearch" :placeholder="canSearch ? t('search.placeholder') : t('search.noSource')" autofocus @input="onQuery" />
          <button v-if="query" type="button" class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground" :aria-label="t('search.clear')" @click="clear"><X class="size-4" /></button>
        </div>
        <Button type="submit" :disabled="!canSearch || !query.trim() || searching">{{ t('search.go') }}</Button>
      </form>
      <div class="flex flex-wrap items-center gap-1.5" role="tablist" :aria-label="t('search.sources')">
        <span class="mr-1 text-xs text-muted-foreground">{{ t('search.sources') }}</span>
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
    </CardHeader>
    <CardContent>
      <p v-if="!canSearch" class="text-sm text-muted-foreground">{{ t('search.noSource') }}</p>

      <!-- results for the submitted query -->
      <template v-else-if="searched">
        <div class="mb-2 flex items-center justify-between gap-2">
          <p class="truncate text-xs text-muted-foreground">
            {{ t('search.results', { q: searched }) }}<template v-if="!searching"> · {{ results.length }}</template>
          </p>
          <Button variant="ghost" size="xs" @click="clear"><X />{{ t('search.clear') }}</Button>
        </div>
        <div v-if="searching" class="space-y-3">
          <div v-for="i in 3" :key="i" class="flex items-center gap-3">
            <Skeleton class="size-11 rounded-md" />
            <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
          </div>
        </div>
        <div v-else-if="results.length" class="divide-y">
          <TrackRow v-for="tr in results" :key="tr.id" :track="tr" hearts>
            <Button size="sm" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
          </TrackRow>
        </div>
        <p v-else class="text-sm text-muted-foreground">{{ t('search.empty') }}</p>
      </template>

      <!-- browse: chart / most played / favorites -->
      <template v-else>
        <div class="mb-2 flex items-center gap-1.5" role="tablist">
          <Button v-for="tab in tabs" :key="tab.id" size="xs" :variant="browse === tab.id ? 'secondary' : 'ghost'" role="tab" :aria-selected="browse === tab.id" @click="browse = tab.id">{{ tab.label }}</Button>
        </div>
        <FolderView v-if="browse === 'dir'" :source="selected" id="" />
        <div v-else-if="browse === 'chart' && chartLoading" class="space-y-3">
          <div v-for="i in 4" :key="i" class="flex items-center gap-3">
            <Skeleton class="size-11 rounded-md" />
            <div class="flex-1 space-y-2"><Skeleton class="h-3 w-2/3" /><Skeleton class="h-3 w-1/3" /></div>
          </div>
        </div>
        <div v-else-if="list.length" class="divide-y">
          <TrackRow v-for="(tr, i) in list" :key="`${tr.source}:${tr.id}`" :track="tr" :index="i + 1" hearts>
            <Button size="sm" :disabled="!isEnabled(tr.source)" :title="isEnabled(tr.source) ? '' : t('search.sourceOff', { name: tr.source })" @click="request(tr)"><Plus />{{ t('search.request') }}</Button>
          </TrackRow>
        </div>
        <p v-else class="text-sm text-muted-foreground">{{ emptyText }}</p>
      </template>
    </CardContent>
  </Card>
</template>
