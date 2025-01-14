<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0-beta3/css/all.min.css">
<template>
  <aside class="main-sidebar elevation-4" data-theme-default="dark">
    <router-link to="/admin/home" class="brand-link">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" class="icon-monitor" width="100%" height="100%">
        <path fill="#B197FC"
              d="M256 16C123.5 16 16 123.5 16 256S123.5 496 256 496 496 388.5 496 256 388.5 16 256 16zM121.7 429.1C70.1 389 36.7 326.3 36.7 256a218.5 218.5 0 0 1 9.6-64.1l102.9-17.9-.1 11-13.9 2s-.1 12.5-.1 19.5a12.8 12.8 0 0 0 4.9 10.3l9.5 7.4zm105.7-283.3 8.5-7.6s6.9-5.4-.1-9.3c-7.2-4-39.5-34.5-39.5-34.5-5.3-5.5-8.3-7.3-15.5 0 0 0-32.3 30.5-39.5 34.5-7.1 4-.1 9.3-.1 9.3l8.5 7.6 0 4.4L76 131c39.6-56.9 105.5-94.3 180-94.3A218.8 218.8 0 0 1 420.9 111.8l-193.5 37.7zm34.1 329.3-33.9-250.9 9.5-7.4a12.8 12.8 0 0 0 4.9-10.3c0-7-.1-19.5-.1-19.5l-13.9-2-.1-10.5 241.7 31.4A218.9 218.9 0 0 1 475.3 256C475.3 375.1 379.8 472.2 261.4 475.1z"/>
      </svg>
      <span>BeanQ Monitor</span>
    </router-link>
    <div class="sidebar" style="">
      <nav class="mt-2">
        <ul id="sidebar" class="nav nav-sidebar flex-column nav-flat" role="menu" data-accordion="false">
          <li v-for="(item,key) in nodes" :key="key" class="nav-item" :class="activeNodeId === item.NodeId ? 'active' : ''">
            <a class="nav-link" @click="chooseNode(item)" href="javascript:;">
              {{ item.Master }}<br/>
              <span style="font-size: 14px">{{item.Ip}}</span>
            </a>
          </li>
        </ul>
      </nav>
    </div>
  </aside>
</template>

<script setup>

import {useRoute} from 'vueRouter';
import {ref, toRefs, onMounted, watch, reactive} from "vue";

const data = reactive({
  nodes: [],
  activeNodeId: ""
})

const route = ref('/admin/home');

const uroute = useRoute();


onMounted(async () => {

  const nodes = await dashboardApi.Nodes();
  data.nodes = nodes.data;

  let nodeId = sessionStorage.getItem("nodeId");
  if (nodeId === "") {
    nodeId = nodes.data[0].NodeId;
  }
  data.activeNodeId = nodeId;
  sessionStorage.setItem("nodeId", nodeId);

  route.value = uroute.fullPath;
})
watch(() => uroute.fullPath, (newVal, oldVal) => {
  route.value = newVal;
})

function chooseNode(item) {
  sessionStorage.setItem("nodeId", item.NodeId);
  window.href.reload();
}

const {nodes, activeNodeId} = toRefs(data);

</script>

<style scoped>

.main-sidebar {
  display: flex;
  flex-direction: column;
  bottom: 0;
  float: none;
  position: fixed;
  background-color: #343a40;
  height: 100vh;
  overflow-y: hidden;
  z-index: 1038;
  transition: margin-left .3s ease-in-out, width .3s ease-in-out;
  width: 15vw;
}
.elevation-4 {
  box-shadow: 0 14px 28px rgba(0, 0, 0, .25), 0 10px 10px rgba(0, 0, 0, .22) !important;
}
.main-sidebar a {
  text-decoration: none !important;
}
.icon-monitor {
  width: 2rem;
  margin-right: 0.5rem;
}
.brand-link {
  color: #B197FC;
  border-bottom: 1px solid #4b545c;
  display: flex;
  align-items: center;
  font-size: 1.15rem;
  padding: 1.35rem .5rem;
  transition: width .3s ease-in-out;
  white-space: nowrap;
  overflow: hidden;
}

.sidebar {
  height: calc(100vh - (3.5rem + 1px));
  overflow-x: hidden;
  overflow-y: auto;
  padding: 0 .5rem;
}

.nav-flat {
  margin: -.25rem -.5rem 0;
}

.sidebar .nav-link {
  color: #c2c7d0;
  font-size: 1.05rem;
}
.sidebar .nav-item a:hover {
  color: #fff !important;
}
.sidebar .nav-sidebar .active {
  background-color: #3d9970;
}
.sidebar .nav-sidebar .active a {
   color: #fff !important;
 }

</style>

