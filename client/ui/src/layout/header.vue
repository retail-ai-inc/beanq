<template>
  <nav class="navbar navbar-expand-lg bg-body-tertiary">
    <div class="container-fluid">
      <router-link to="/admin/home" class="navbar-brand">Beanq Monitor</router-link>

      <div class="collapse navbar-collapse" id="navbarSupportedContent">
        <ul class="navbar-nav me-auto mb-2 mb-lg-0">
          <li class="nav-item">
            <router-link to="/admin/home" class="nav-link text-muted" :class="route == '/admin/home' ? 'active' : ''">Home</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/admin/server" class="nav-link text-muted" :class="route == '/admin/server' ? 'active' : ''">Server</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/admin/schedule" class="nav-link text-muted" :class="route == '/admin/schedule' ? 'active' : ''">Schedule</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/admin/queue" class="nav-link text-muted" :class="route == '/admin/queue' ? 'active' : ''">Queue</router-link>
          </li>
          <li class="nav-item dropdown">

            <a class="nav-link dropdown-toggle text-muted" :class="route == '/admin/log/success' || route == '/admin/log/error' ? 'active' : ''"  role="button" data-bs-toggle="dropdown" aria-expanded="false">
              Log
            </a>
            <ul class="dropdown-menu dropdown-menu-dark" >
              <li>
                <router-link to="/admin/log/success" class="dropdown-item nav-link text-muted" :class="route=='/admin/log/success' ? 'active' : ''">Success Log</router-link>
              </li>
              <li>
                <router-link to="/admin/log/error" class="dropdown-item nav-link text-muted" :class="route == '/admin/log/error' ? 'active' : ''">Error Log</router-link>
              </li>
            </ul>

          </li>
          <li class="nav-item">
            <router-link to="/admin/redis" class="nav-link text-muted" :class="route == '/admin/redis' ? 'active' : ''">Redis</router-link>
          </li>
        </ul>
        <span class="navbar-text" style="color:#fff">
          <div class="dropdown">
            <button class="btn btn-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false" style="background: #212529;border: none;">
              Setting
            </button>
            <ul class="dropdown-menu">
              <li><a class="dropdown-item" @click="setting">Setting</a></li>
              <li><a class="dropdown-item" @click="logout">Logout</a></li>
            </ul>
          </div>
        </span>

      </div>
    </div>
  </nav>
</template>


<script setup>

import { useRoute,useRouter } from 'vueRouter';
import {ref,onMounted,watch} from "vue";

const route = ref('/admin/home');

const uroute = useRoute();
const urouter = useRouter();


onMounted(()=>{
  route.value = uroute.fullPath;
})
watch(()=>uroute.fullPath,(newVal,oldVal)=>{
  route.value = newVal;
})
function setting(){

}
function logout(){
  sessionStorage.clear();
  urouter.push("/login");
}
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
.navbar-text .btn-secondary:focus{
  border: none !important;
}
.example {
  color: v-bind('color');
}
</style>

