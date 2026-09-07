<script setup lang="ts">
import Artwork from './Artwork.vue'
import SourceLink from './SourceLink.vue'
import type { Track } from './types'
import { fmtDuration } from './types'

defineProps<{ track: Track; index?: number; subtitle?: string }>()
</script>

<template>
  <div class="flex items-center gap-3 py-2">
    <span v-if="index !== undefined" class="w-5 text-right font-mono text-xs text-muted-foreground">{{ index }}</span>
    <Artwork :src="track.artworkUrl" :alt="track.album" class="size-11" />
    <div class="min-w-0 flex-1">
      <p class="truncate text-sm font-medium">{{ track.title }}</p>
      <p class="truncate text-xs text-muted-foreground">
        {{ track.artist || '—' }}<span v-if="subtitle"> · {{ subtitle }}</span>
      </p>
      <SourceLink v-if="track.externalUrl" :href="track.externalUrl" />
    </div>
    <span class="font-mono text-xs text-muted-foreground tabular-nums">{{ fmtDuration(track.duration) }}</span>
    <slot />
  </div>
</template>
