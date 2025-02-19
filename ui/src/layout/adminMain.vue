<template>
  <div class="wrapper">
    <HeaderLayout :hlang="hlang" :nav="nav" @action="action"></HeaderLayout>
    <SidebarLayout></SidebarLayout>
    <ContentLayout></ContentLayout>
  </div>
</template>

<script setup>
import {ref, provide,onMounted} from "vue";

import HeaderLayout from "./header.vue";
import SidebarLayout from "./sidebar.vue";
import ContentLayout from "./content.vue";

const [nav,hlang,flag] = [ref(Nav),ref(Langs),ref("en")];
provide("i18n",flag);

function action(item){
  flag.value = item.flag;
}

onMounted(()=>{
  flag.value = sessionStorage.getItem("i18n") || "en";
})

</script>