<template>
  <div class="wrapper">
    <HeaderLayout :hlang="hlang" @action="action"></HeaderLayout>
    <SidebarLayout></SidebarLayout>
    <ContentLayout></ContentLayout>
  </div>
</template>

<script setup>
import {ref, provide} from "vue";

import HeaderLayout from "./header.vue";
import SidebarLayout from "./sidebar.vue";
import ContentLayout from "./content.vue";

const hlang = ref(Base.GetLang(I18n));

function action(item){
  hlang.value = item;
}

provide("i18n",hlang);
</script>