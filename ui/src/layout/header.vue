<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0-beta3/css/all.min.css">
<template>
  <nav class="main-header navbar navbar-expand navbar-danger navbar-dark">
    <div class="container-fluid">
      <ul class="navbar-nav w-100">
        <li class="nav-item">
          <router-link to="/admin/home" class="nav-link" :class="route === '/admin/home' ? 'active' : ''">
            Home
          </router-link>
        </li>
        <li class="nav-item">
          <router-link to="/admin/schedule" class="nav-link" :class="route === '/admin/schedule' ? 'active' : ''">
            Schedule
          </router-link>
        </li>
        <li class="nav-item">
          <router-link to="/admin/queue" class="nav-link" :class="route === '/admin/queue' ? 'active' : ''">
            Channel
          </router-link>
        </li>
        <li class="nav-item dropdown">
          <a class="nav-link dropdown-toggle" :class="route === '/admin/log/event' || route === '/admin/log/dlq' || route === '/admin/log/workflow' ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">
            Log
          </a>
          <ul class="dropdown-menu dropdown-menu-color">
            <li>
              <router-link to="/admin/log/event" class="dropdown-item" :class="route ==='/admin/log/event' ? 'active' : ''">
                EventLog
              </router-link>
            </li>
            <li>
              <router-link to="/admin/log/dlq" class="dropdown-item" :class="route === '/admin/log/dlq' ? 'active' : ''">
                DLQLog
              </router-link>
            </li>
            <li>
              <router-link to="/admin/log/workflow" class="dropdown-item" :class="route === '/admin/log/workflow' ? 'active' : ''">
                WorkFlowLog
              </router-link>
            </li>
          </ul>
        </li>
        <li class="nav-item dropdown">
          <a class="nav-link dropdown-toggle" :class="route==='/admin/redis' || route === '/admin/redis/monitor' ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">
            Redis
          </a>
          <ul class="dropdown-menu dropdown-menu-color">
            <li>
              <router-link to="/admin/redis" class="dropdown-item " :class="route==='/admin/redis' ? 'active' : ''">
                Info
              </router-link>
            </li>
            <li>
              <router-link to="/admin/redis/monitor" class="dropdown-item" :class="route === '/admin/redis/monitor' ? 'active' : ''">
                Command
              </router-link>
            </li>
          </ul>
        </li>
        <li class="nav-item dropdown">
          <a class="nav-link dropdown-toggle" role="button" data-bs-toggle="dropdown" aria-expanded="false">
            Setting
          </a>
          <ul class="dropdown-menu dropdown-menu-color dropdown-menu-end">
            <li>
              <router-link to="/admin/optLog" class="dropdown-item" :class="route==='/admin/optLog' ? 'active' : ''">
                Operation Log
              </router-link>
            </li>
            <li>
              <router-link to="/admin/user" class="dropdown-item" :class="route==='/admin/user' ? 'active' : ''">
                User
              </router-link>
            </li>
            <li>
              <router-link to="/login" class="dropdown-item">
                Logout
              </router-link>
            </li>
<!--            <li><a class="dropdown-item" @click="optLog" href="javascript:;">Operation Log</a></li>-->
<!--            <li><a class="dropdown-item" @click="userList" href="javascript:;">User</a></li>-->
<!--            <li><a class="dropdown-item" @click="logout" href="javascript:;">Logout</a></li>-->
          </ul>
        </li>
      </ul>
    </div>
  </nav>
</template>

<script setup>

import {useRoute, useRouter} from 'vueRouter';
import {ref, toRefs, onMounted, watch, reactive} from "vue";

const data = reactive({
  nodes: [],
  activeNodeId: ""
})

const route = ref('/admin/home');

const uroute = useRoute();
const urouter = useRouter();


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

function optLog() {
  urouter.push("/admin/optLog");
}

function userList() {
  urouter.push("/admin/user")
}

function logout() {
  sessionStorage.clear();
  urouter.push("/login");
}

function chooseNode(item) {
  sessionStorage.setItem("nodeId", item.NodeId);
  window.href.reload();
}

const {nodes, activeNodeId} = toRefs(data);

</script>

<style scoped>

.main-header {
  transition: margin-left .3s ease-in-out;
  margin-left: 250px;
  background-color: #B197FC;
  border-bottom: 1px solid #4b545c;
  z-index: 1034;
  color: #ffffff;
}
.navbar {
  font-size: 1.25rem;
  position: relative;
  display: flex;
  align-items: center;
  padding: .5rem .5rem;
}
.navbar-expand {
  flex-flow: row nowrap;
  justify-content: flex-start;
}

.navbar-nav li:last-child {
  margin-left: auto;
}

.navbar .navbar-nav .nav-item .active {
  color: #000000 !important;
}
.navbar .navbar-nav .nav-item a:hover {
  color: #000000 !important;
}

.dropdown-item.active {
  text-decoration: none;
  background-color: #B197FC;;
}

.dropdown-item:active {
  background-color: #B197FC;;
}

.nav-link {
  color: #ffffff !important;
}

.dropdown-menu-color {
  color: #000000;
  background-color: #ffffff;
  border-color:#ffffff;
}

</style>

