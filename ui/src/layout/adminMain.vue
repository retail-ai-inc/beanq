<template>
  <div class="wrapper">
    <HeaderLayout></HeaderLayout>
    <SidebarLayout></SidebarLayout>
    <ContentLayout></ContentLayout>
  </div>
</template>

<script setup>
import HeaderLayout from "./header.vue";
import SidebarLayout from "./sidebar.vue";
import ContentLayout from "./content.vue";
</script>