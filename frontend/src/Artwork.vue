<script setup lang="ts">
// Album art with a music-note fallback. Spotify art must stay uncropped with
// rounded corners (design guidelines); object-cover on a square keeps it whole.
import { Music } from '@lucide/vue'
defineProps<{ src?: string; alt?: string; class?: string }>()
</script>

<template>
  <div :class="['relative shrink-0 overflow-hidden rounded-md bg-muted', $props.class]">
    <img v-if="src" :src="src" :alt="alt ?? ''" class="size-full object-cover" />
    <Music v-else class="absolute inset-0 m-auto size-1/3 text-muted-foreground/60" />
  </div>
</template>
