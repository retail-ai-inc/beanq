<template>
  <router-view v-slot="{Component}">
    <component :is="Component"/>
  </router-view>
</template>

<script setup>
// import { useRoute } from 'vueRouter';
// import { ref, watch, onMounted } from "vue";
//
// const route = ref('/');
//
// const useR = useRoute();
// let fullPath = useR.fullPath;
// fullPath = fullPath.replace("/", "");
// fullPath = fullPath.slice(0, 1).toUpperCase() + fullPath.slice(1).toLowerCase();
//
// onMounted(() => {
//   route.value = fullPath;
// })
// watch(() => useR.fullPath, (newVal, oldVal) => {
//   let newV = newVal.replace("/", "");
//   newV = newV.slice(0, 1).toUpperCase() + newV.slice(1).toLowerCase();
//   route.value = newV;
// })
</script>
