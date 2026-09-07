<script setup lang="ts">
import { locale, setLocale } from './i18n'
</script>

<template>
  <button class="font-mono text-xs text-mute hover:text-ink" @click="setLocale(locale === 'en' ? 'ko' : 'en')" :aria-label="locale === 'en' ? '한국어' : 'English'">
    {{ locale === 'en' ? 'KO' : 'EN' }}
  </button>
</template>
