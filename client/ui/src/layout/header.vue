<template>
  <nav class="navbar navbar-expand-lg bg-body-tertiary">
    <div class="container-fluid">
      <router-link to="/" class="navbar-brand">Beanq Monitor</router-link>

      <div class="collapse navbar-collapse" id="navbarSupportedContent">
        <ul class="navbar-nav me-auto mb-2 mb-lg-0">
          <li class="nav-item">
            <router-link to="/" class="nav-link text-muted" :class="route == '/' ? 'active' : ''">Home</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/server" class="nav-link text-muted" :class="route == '/server' ? 'active' : ''">Server</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/schedule" class="nav-link text-muted" :class="route == '/schedule' ? 'active' : ''">Schedule</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/queue" class="nav-link text-muted" :class="route == '/queue' ? 'active' : ''">Queue</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/log" class="nav-link text-muted" :class="route == '/log' ? 'active' : ''">Log</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/redis" class="nav-link text-muted" :class="route == '/redis' ? 'active' : ''">Redis</router-link>
          </li>
        </ul>
      </div>
    </div>
  </nav>
</template>


<script setup>

import { useRoute } from 'vueRouter';
import {ref,onMounted,watch} from "vue";

const route = ref('/');

const useR = useRoute();

onMounted(()=>{
  route.value = useR.fullPath;
})
watch(()=>useR.fullPath,(newVal,oldVal)=>{
  route.value = newVal;
})

</script>

<style scoped>
.navbar{
  background-color: var(--bs-body-color);
}
.navbar .navbar-brand{
  color:var(--bs-body-bg)
}
.navbar .navbar-nav .nav-item .active{
  color:#ffcd39 !important
}
.navbar .navbar-nav .nav-item a:hover{
  color:#ffcd39 !important
}
.example {
  color: v-bind('color');
}
</style>

