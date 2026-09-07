<script setup lang="ts">
import { Heart } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import Artwork from './Artwork.vue'
import SourceIcon from './SourceIcon.vue'
import SourceLink from './SourceLink.vue'
import { isFav, toggleFav } from './favorites'
import { t } from './i18n'
import type { Track } from './types'
import { fmtDuration } from './types'

// Artist and album become links when the source gave them browse keys.
defineProps<{ track: Track; index?: number; subtitle?: string; hearts?: boolean }>()
</script>

<template>
  <div class="flex items-center gap-3 py-2">
    <span v-if="index !== undefined" class="w-5 text-right font-mono text-xs text-muted-foreground">{{ index }}</span>
    <Artwork :src="track.artworkUrl" :alt="track.album" class="size-11" />
    <div class="min-w-0 flex-1">
      <p class="truncate text-sm font-medium">{{ track.title }}</p>
      <p class="flex items-center gap-1 truncate text-xs text-muted-foreground">
        <SourceIcon :source="track.source" class="size-3" />
        <span class="truncate">
          <RouterLink v-if="track.artistId" :to="`/artist/${track.source}/${track.artistId}`" class="hover:underline">{{ track.artist }}</RouterLink>
          <template v-else>{{ track.artist || '—' }}</template>
          <template v-if="track.album"> · <RouterLink v-if="track.albumId" :to="`/album/${track.source}/${track.albumId}`" class="hover:underline">{{ track.album }}</RouterLink><template v-else>{{ track.album }}</template></template>
          <template v-if="track.year"> · {{ track.year }}</template>
          <template v-if="subtitle"> · {{ subtitle }}</template>
        </span>
      </p>
      <SourceLink v-if="track.externalUrl" :href="track.externalUrl" />
    </div>
    <span class="font-mono text-xs text-muted-foreground tabular-nums">{{ fmtDuration(track.duration) }}</span>
    <Button v-if="hearts" variant="ghost" size="icon-sm" :aria-label="t(isFav(track) ? 'fav.remove' : 'fav.add')" :title="t(isFav(track) ? 'fav.remove' : 'fav.add')" @click="toggleFav(track)">
      <Heart :class="isFav(track) ? 'fill-current' : ''" />
    </Button>
    <slot />
  </div>
</template>
