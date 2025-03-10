<template>
  <div class="wrapper">
    <HeaderLayout :hlang="hlang" :nav="nav" @action="action"></HeaderLayout>
    <SidebarLayout></SidebarLayout>
    <ContentLayout></ContentLayout>
  </div>
</template>

<script setup>
import {ref,onMounted} from "vue";

import HeaderLayout from "./header.vue";
import SidebarLayout from "./sidebar.vue";
import ContentLayout from "./content.vue";

const [nav,hlang,] = [ref(Nav),ref(Langs)];

function action(item){

}

onMounted(()=>{

})

</script>