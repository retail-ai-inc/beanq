<template>
    <div class="container-fluid d-flex flex-column">
        <headerLayout></headerLayout>
        <div class="my-container">

            <nav aria-label="breadcrumb">
                <ol class="breadcrumb">
                    <li class="breadcrumb-item"><a href="#">Home</a></li>
                    <li class="breadcrumb-item active" aria-current="page">{{ route }}</li>
                </ol>
            </nav>
            <router-view v-slot="{Component}">
                <component :is="Component" />
            </router-view>
        </div>
    </div>
</template>

<script setup>

import headerLayout from "./header.vue";
import { useRoute } from 'vueRouter';
import { ref, watch, onMounted } from "vue";

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
    height: 100%;
    margin: 20px;
    border: 1px solid #f8f9fa;
    border-radius: 5px;
    padding: 15px;
    background-color: #f8f9fa;
}
</style>