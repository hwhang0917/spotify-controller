<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { locale, setLocale } from './i18n'
</script>

<template>
  <Button variant="ghost" size="xs" class="font-mono text-muted-foreground" @click="setLocale(locale === 'en' ? 'ko' : 'en')">
    {{ locale === 'en' ? 'KO' : 'EN' }}
  </Button>
</template>
