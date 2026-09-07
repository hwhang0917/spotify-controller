<script setup lang="ts">
import { onMounted, ref } from 'vue'

const status = ref<'checking' | 'ok' | 'down'>('checking')

onMounted(async () => {
  try {
    const res = await fetch('/api/health')
    status.value = res.ok ? 'ok' : 'down'
  } catch {
    status.value = 'down'
  }
})
</script>

<template>
  <main class="min-h-screen bg-neutral-950 text-neutral-100 flex flex-col items-center justify-center gap-4 p-8">
    <h1 class="text-4xl font-bold tracking-tight">vibe-music</h1>
    <p class="text-neutral-400">Guest UI</p>
    <p class="text-sm">
      server:
      <span :class="status === 'ok' ? 'text-emerald-400' : 'text-rose-400'">{{ status }}</span>
    </p>
  </main>
</template>
