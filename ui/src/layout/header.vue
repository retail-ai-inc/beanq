<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0-beta3/css/all.min.css">
<template>
  <nav class="navbar navbar-expand-lg bg-body-tertiary">
    <div class="container-fluid">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" class="icon-monitor">
        <path fill="#B197FC" d="M256 16C123.5 16 16 123.5 16 256S123.5 496 256 496 496 388.5 496 256 388.5 16 256 16zM121.7 429.1C70.1 389 36.7 326.3 36.7 256a218.5 218.5 0 0 1 9.6-64.1l102.9-17.9-.1 11-13.9 2s-.1 12.5-.1 19.5a12.8 12.8 0 0 0 4.9 10.3l9.5 7.4zm105.7-283.3 8.5-7.6s6.9-5.4-.1-9.3c-7.2-4-39.5-34.5-39.5-34.5-5.3-5.5-8.3-7.3-15.5 0 0 0-32.3 30.5-39.5 34.5-7.1 4-.1 9.3-.1 9.3l8.5 7.6 0 4.4L76 131c39.6-56.9 105.5-94.3 180-94.3A218.8 218.8 0 0 1 420.9 111.8l-193.5 37.7zm34.1 329.3-33.9-250.9 9.5-7.4a12.8 12.8 0 0 0 4.9-10.3c0-7-.1-19.5-.1-19.5l-13.9-2-.1-10.5 241.7 31.4A218.9 218.9 0 0 1 475.3 256C475.3 375.1 379.8 472.2 261.4 475.1z"/>
      </svg>
      <router-link to="/admin/home" class="navbar-brand">BeanQ Monitor</router-link>
      <div class="collapse navbar-collapse" id="navbarSupportedContent">
        <ul class="navbar-nav me-auto mb-2 mb-lg-0">
          <li class="nav-item">
            <router-link to="/admin/home" class="nav-link text-muted" :class="route == '/admin/home' ? 'active' : ''">Home</router-link>
          </li>
<!--          <li class="nav-item">-->
<!--            <router-link to="/admin/server" class="nav-link text-muted" :class="route == '/admin/server' ? 'active' : ''">Server</router-link>-->
<!--          </li>-->
          <li class="nav-item">
            <router-link to="/admin/schedule" class="nav-link text-muted" :class="route == '/admin/schedule' ? 'active' : ''">Schedule</router-link>
          </li>
          <li class="nav-item">
            <router-link to="/admin/queue" class="nav-link text-muted" :class="route == '/admin/queue' ? 'active' : ''">Channel</router-link>
          </li>
          <li class="nav-item dropdown">
            <a class="nav-link dropdown-toggle text-muted"
               :class="route == '/admin/log/event' || route == '/admin/log/dlq' || route == '/admin/log/workflow' ? 'active' : ''"
               role="button"
               data-bs-toggle="dropdown"
               aria-expanded="false">
              Log
            </a>
            <ul class="dropdown-menu dropdown-menu-dark">
              <li>
                <router-link to="/admin/log/event"
                             class="dropdown-item nav-link text-muted"
                             :class="route=='/admin/log/event' ? 'active' : ''">
                  EventLog
                </router-link>
              </li>
              <li>
                <router-link to="/admin/log/dlq"
                             class="dropdown-item nav-link text-muted"
                             :class="route == '/admin/log/dlq' ? 'active' : ''">
                  DLQLog
                </router-link>
              </li>
              <li>
                <router-link to="/admin/log/workflow"
                             class="dropdown-item nav-link text-muted"
                             :class="route == '/admin/log/workflow' ? 'active' : ''">
                  WorkFlowLog
                </router-link>
              </li>
            </ul>
          </li>
          <li class="nav-item dropdown">
            <a class="nav-link dropdown-toggle text-muted" :class="route=='/admin/redis' || route == '/admin/redis/monitor' ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">Redis</a>
            <ul class="dropdown-menu dropdown-menu-dark">
              <li>
                <router-link to="/admin/redis" class="dropdown-item nav-link text-muted" :class="route=='/admin/redis' ? 'active' : ''">Info</router-link>
              </li>
              <li>
                <router-link to="/admin/redis/monitor" class="dropdown-item nav-link text-muted" :class="route == '/admin/redis/monitor' ? 'active' : ''">Command</router-link>
              </li>
            </ul>
          </li>
        </ul>
        <span class="navbar-text" style="color:#fff">
          <div class="dropdown">
            <button class="btn btn-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false" style="background: #212529;border: none;">
              Setting
            </button>
            <ul class="dropdown-menu">
              <li><a class="dropdown-item" @click="userList" href="javascript:;">User</a></li>
              <li><a class="dropdown-item" @click="logout" href="javascript:;">Logout</a></li>
            </ul>
          </div>
        </span>
      </div>
    </div>
  </nav>
</template>

<script setup>

import {useRoute, useRouter} from 'vueRouter';
import {ref, onMounted, watch} from "vue";

const route = ref('/admin/home');

const uroute = useRoute();
const urouter = useRouter();


onMounted(() => {
  route.value = uroute.fullPath;
})
watch(() => uroute.fullPath, (newVal, oldVal) => {
  route.value = newVal;
})

function userList() {
  urouter.push("/admin/user")
}

function logout() {
  sessionStorage.clear();
  urouter.push("/login");
}


</script>

<style scoped>
.navbar {
  background-color: var(--bs-body-color);
}

.icon-monitor {
  width: 28px;
  margin-right: 2px;
}

.navbar .navbar-brand {
  color: #B197FC !important
}

.navbar .navbar-nav .nav-item .active {
  color: #ffffff !important
}

.navbar .navbar-nav .nav-item a:hover {
  color: #ffffff !important
}

.navbar-text .btn-secondary:focus {
  border: none !important;
}

.example {
  color: v-bind('color');
}
</style>

