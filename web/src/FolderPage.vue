<script setup lang="ts">
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import FolderView from './FolderView.vue'
import SourceIcon from './SourceIcon.vue'
import { t } from './i18n'

withDefaults(defineProps<{ source: string; id?: string }>(), { id: '' })
</script>

<template>
  <Card>
    <CardHeader>
      <p class="eyebrow flex items-center gap-1.5"><SourceIcon :source="source" class="size-3" />{{ t('browse.folder') }}</p>
    </CardHeader>
    <CardContent>
      <FolderView :source="source" :id="id" />
    </CardContent>
  </Card>
</template>
