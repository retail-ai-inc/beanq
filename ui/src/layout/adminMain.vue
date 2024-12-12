<template>
  <div class="container-fluid d-flex flex-column">
    <headerLayout></headerLayout>
    <div class="my-container">

      <div class="row">
        <nav aria-label="breadcrumb">
          <ol class="breadcrumb" style="float: right">
            <li class="breadcrumb-item">
              <a href="#">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 576 512" style="width: 22px; padding-bottom: 3px;">
                  <path fill="#B197FC"
                        d="M575.8 255.5c0 18-15 32.1-32 32.1l-32 0 .7 160.2c0 2.7-.2 5.4-.5 8.1l0 16.2c0 22.1-17.9 40-40 40l-16 0c-1.1 0-2.2 0-3.3-.1c-1.4 .1-2.8 .1-4.2 .1L416 512l-24 0c-22.1 0-40-17.9-40-40l0-24 0-64c0-17.7-14.3-32-32-32l-64 0c-17.7 0-32 14.3-32 32l0 64 0 24c0 22.1-17.9 40-40 40l-24 0-31.9 0c-1.5 0-3-.1-4.5-.2c-1.2 .1-2.4 .2-3.6 .2l-16 0c-22.1 0-40-17.9-40-40l0-112c0-.9 0-1.9 .1-2.8l0-69.7-32 0c-18 0-32-14-32-32.1c0-9 3-17 10-24L266.4 8c7-7 15-8 22-8s15 2 21 7L564.8 231.5c8 7 12 15 11 24z"/>
                </svg>
              </a>
            </li>
            <li class="breadcrumb-item active" aria-current="page">{{ route }}</li>
          </ol>
        </nav>
      </div>

      <hr>

      <div class="row">
        <router-view v-slot="{Component}">
          <transition name="fade" mode="out-in">
            <component :is="Component"/>
          </transition>
        </router-view>
      </div>

    </div>
  </div>
</template>

<script setup>

import headerLayout from "./header.vue";
import {useRoute} from 'vueRouter';
import {ref, watch, onMounted} from "vue";

const route = ref('/');

const useR = useRoute();
let fullPath = useR.fullPath;
fullPath = fullPath.replace("/", "");
fullPath = fullPath.slice(0, 1).toUpperCase() + fullPath.slice(1).toLowerCase();

onMounted(() => {
  route.value = fullPath;
})
watch(() => useR.fullPath, (newVal, oldVal) => {
  let newV = newVal.replace("/", "");
  newV = newV.slice(0, 1).toUpperCase() + newV.slice(1).toLowerCase();
  route.value = newV;
})


</script>

<style scoped>
.container-fluid {
  padding: 0;
  height: 100%;
}

.my-container {
  margin: 1.25rem;
  border: 0.0625rem solid #f8f9fa;
  border-radius: 0.3125rem;
  padding: 0.9375rem;
  background-color: #fff;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.5s ease;
}

.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>