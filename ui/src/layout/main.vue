<template>
  <router-view v-slot="{Component}">
    <component :is="Component"/>
    <En/>
    <Ja/>
  </router-view>
</template>

<script setup>
import En from "./i18n/en.vue";
import Ja from "./i18n/ja.vue";
</script>
