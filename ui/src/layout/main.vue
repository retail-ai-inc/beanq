<template>
  <div class="container-fluid d-flex flex-column">

      <router-view v-slot="{Component}">
        <component :is="Component" />
      </router-view>

  </div>
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

<style scoped>
.container-fluid {
  padding: 0;
  height: 100%;
}
</style>