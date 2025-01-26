<template>
  <div class="content-wrapper">
    <section class="content-header">
      <div class="container-fluid">
        <div class="row mb-1">
          <div class="col-sm"></div>
          <div class="col-sm-auto mt-1">
            <div class="float-sm-right">
              <ol class="breadcrumb">
                <li class="breadcrumb-item">
                  <a href="/#/admin/home">
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 576 512" style="width: 22px; padding-bottom: 3px;">
                      <path fill="#B197FC"
                            d="M575.8 255.5c0 18-15 32.1-32 32.1l-32 0 .7 160.2c0 2.7-.2 5.4-.5 8.1l0 16.2c0 22.1-17.9 40-40 40l-16 0c-1.1 0-2.2 0-3.3-.1c-1.4 .1-2.8 .1-4.2 .1L416 512l-24 0c-22.1 0-40-17.9-40-40l0-24 0-64c0-17.7-14.3-32-32-32l-64 0c-17.7 0-32 14.3-32 32l0 64 0 24c0 22.1-17.9 40-40 40l-24 0-31.9 0c-1.5 0-3-.1-4.5-.2c-1.2 .1-2.4 .2-3.6 .2l-16 0c-22.1 0-40-17.9-40-40l0-112c0-.9 0-1.9 .1-2.8l0-69.7-32 0c-18 0-32-14-32-32.1c0-9 3-17 10-24L266.4 8c7-7 15-8 22-8s15 2 21 7L564.8 231.5c8 7 12 15 11 24z"/>
                    </svg>
                  </a>
                </li>
                <li class="breadcrumb-item active" aria-current="page">{{ route }}</li>
              </ol>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="content">
      <div class="container-fluid pb-4">
        <div class="card card-olive card-outline">
          <div class="card-body">
            <div class="row">
              <router-view v-slot="{Component}">
                <transition name="fade" mode="out-in">
                  <component :is="Component"/>
                </transition>
              </router-view>
            </div>
          </div>
        </div>
      </div>
    </section>

  </div>
</template>

<script setup>
import {useRoute} from 'vueRouter';
import {ref, watch, onMounted} from "vue";

const route = ref('/');

const useR = useRoute();
let fullPath = useR.path;
fullPath = fullPath.replace("/", "");
fullPath = fullPath.slice(0, 1).toUpperCase() + fullPath.slice(1).toLowerCase();

onMounted(() => {
  route.value = fullPath;
})
watch(() => useR.path, (newVal, oldVal) => {
  let newV = newVal.replace("/", "");
  newV = newV.slice(0, 1).toUpperCase() + newV.slice(1).toLowerCase();
  route.value = newV;
})
</script>

<style scoped>

.content-wrapper {
  margin-left: 15vw;
  transition: margin-left .3s ease-in-out;
  height: 100%;
  background-color: #f4f6f9;
  .content-header {
    padding: 1rem .5rem;
  }
}

.container-fluid {
  width: 100%;
  padding-right: 8px;
  padding-left: 8px;
  margin-right: auto;
  margin-left: auto;
}
.content-header .breadcrumb {
  display: flex;
  flex-wrap: wrap;
  list-style: none;
  border-radius: .25rem;
  background-color: transparent;
  line-height: 1.8rem;
  margin-bottom: 0;
  padding: 0;
}

.content {
  padding: 0 .5rem;
}
</style>