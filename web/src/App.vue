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
  <main class="min-h-screen flex items-center justify-center p-6">
    <section class="card w-full max-w-sm space-y-3">
      <p class="eyebrow">guest</p>
      <h1 class="text-2xl font-semibold tracking-tight">vibe-music</h1>
      <p class="text-sm text-body">
        server
        <span class="font-mono" :class="status === 'ok' ? 'text-link' : 'text-error'">{{ status }}</span>
      </p>
    </section>
  </main>
</template>
