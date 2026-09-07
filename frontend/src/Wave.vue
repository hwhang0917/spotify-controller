<script setup lang="ts">
// Five-bar "now playing" wave. Animates while playing, freezes and fades when not.
defineProps<{ playing: boolean }>()
</script>

<template>
  <span class="vm-eq text-foreground" :class="{ playing }" aria-hidden="true"><i /><i /><i /><i /><i /></span>
</template>
