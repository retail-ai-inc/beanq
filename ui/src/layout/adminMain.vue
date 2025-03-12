<template>
  <div class="wrapper">
    <HeaderLayout :hlang="hlang" :nav="nav" @action="action"></HeaderLayout>
    <SidebarLayout></SidebarLayout>
    <ContentLayout></ContentLayout>
    <En/>
    <Ja/>
  </div>
</template>

<script setup>
import {ref,onMounted} from "vue";

import HeaderLayout from "./header.vue";
import SidebarLayout from "./sidebar.vue";
import ContentLayout from "./content.vue";
import En from "./i18n/en.vue";
import Ja from "./i18n/ja.vue";

const [nav,hlang,] = [ref(Nav),ref(Langs)];

function action(item){

}

onMounted(()=>{

})

</script>