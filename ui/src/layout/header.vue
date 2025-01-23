<template>
  <nav class="main-header navbar navbar-expand navbar-danger navbar-dark">
    <div style="width: 1.5rem;height:1.5rem" @click="expand">
        <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-list" viewBox="0 0 16 16">
          <path fill-rule="evenodd" d="M2.5 12a.5.5 0 0 1 .5-.5h10a.5.5 0 0 1 0 1H3a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0 1 .5-.5h10a.5.5 0 0 1 0 1H3a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0 1 .5-.5h10a.5.5 0 0 1 0 1H3a.5.5 0 0 1-.5-.5z"/>
        </svg>
    </div>
    <div class="container-fluid">
      <ul class="navbar-nav">
        <li class="nav-item" v-for="(item,key) in lang.nav" :key="key" :class="item.sub.length > 0 ?'dropdown':''">
          <div v-if="item.sub.length > 0">
            <!--nav sub-->
            <a class="nav-link dropdown-toggle" :class=" item.tos.indexOf(route) !== -1 ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">
              {{item.label}}
            </a>
            <ul class="dropdown-menu dropdown-menu-color">
              <li v-for="(val,ind) in item.sub" :key="ind">
                <router-link :to="val.to" class="dropdown-item" :class="route === val.to ? 'active' : ''">
                  {{val.label}}
                </router-link>
              </li>
            </ul>
          </div>
          <div v-else>
            <!--nav no sub-->
            <router-link v-if="item.sub.length <= 0" :to="item.to" class="nav-link" :class="route === item.to ? 'active' : ''">
              {{item.label}}
            </router-link>
          </div>
        </li>
      </ul>
      <ul class="navbar-nav">
        <li class="nav-item" v-for="(item,key) in lang.setting" :key="key" :class="item.sub.length > 0 ?'dropdown':''">
          <!--nav sub-->
          <div v-if="item.sub.length > 0">

            <div v-if="item.label === 'Language'">
              <a class="nav-link dropdown-toggle" :class="item.tos.indexOf(route) !== -1 ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                {{language}}
              </a>
              <ul class="dropdown-menu dropdown-menu-color">
                <li v-for="(val,ind) in item.sub" :key="ind">
<!--                  <a  class="dropdown-item" :class="route === val.to ? 'active' : ''" @click="chooseLang(val)">-->
                  <a  class="dropdown-item" :class="route === val.to ? 'active' : ''" @click="action(val)">
                    {{val.label}}
                  </a>
                </li>
              </ul>
            </div>
            <div v-else>
              <a class="nav-link dropdown-toggle" :class=" item.tos.indexOf(route) !== -1 ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                {{item.label}}
              </a>
              <ul class="dropdown-menu dropdown-menu-color" style="left: inherit;right: 0;">
                <li v-for="(val,ind) in item.sub" :key="ind">
                  <router-link :to="val.to" class="dropdown-item" :class="route === val.to ? 'active' : ''" v-if="val.label === 'Operation Log'" @click="optLog">
                    {{val.label}}
                  </router-link>
                  <router-link :to="val.to" class="dropdown-item" :class="route === val.to ? 'active' : ''" v-if="val.label === 'User'" @click="userList">
                    {{val.label}}
                  </router-link>
                  <router-link :to="val.to" class="dropdown-item" :class="route === val.to ? 'active' : ''" v-if="val.label === 'Logout'" @click="logout">
                    {{val.label}}
                  </router-link>
                </li>
              </ul>
            </div>
          </div>
          <div v-else>
            <!--nav no sub-->
            <router-link v-if="item.sub.length <= 0" :to="item.to" class="nav-link" :class="route === item.to ? 'active' : ''">
              {{item.label}}
            </router-link>
          </div>
        </li>
      </ul>
    </div>
  </nav>
</template>

<script setup>

import {useRoute, useRouter} from 'vueRouter';
import {ref, toRefs, onMounted, watch, reactive,defineProps,defineEmits} from "vue";

const props = defineProps({
  hlang:{},
})

const emits = defineEmits(['action']);
const action = function (obj){

  if(obj.index === 1){
    language.value = "日本語 (Japanese)";
  }else{
    language.value = "English";
  }
  let index = obj.index;
  sessionStorage.setItem("lang",index);
  lang.value = Base.GetLang(I18n);

  emits("action",lang.value);
}

const data = reactive({
  nodes: [],
  activeNodeId: "",
  isSide:false,
  //lang:{}
})
const lang = ref(props.hlang);

function chooseLang(obj){
  if(obj.index === 1){
    language.value = "日本語 (Japanese)";
  }else{
    language.value = "English";
  }
  let index = obj.index;
  sessionStorage.setItem("lang",index);
  lang.value = Base.GetLang(I18n);
}

const [route,uroute,urouter,language] = [ref("/admin/home"),useRoute(),useRouter(),ref("English")];

function expand(){

  data.isSide = !data.isSide;

  let sideWidth = "calc(15vw - 180px)";
  if(!data.isSide){
     sideWidth = "calc(15vw)";
  }
  let sideBarDom = document.getElementsByClassName("main-sidebar")[0];
  sideBarDom.style.width = sideWidth;

  [
      document.getElementsByClassName("content-wrapper")[0],
      document.getElementsByClassName("main-header")[0]
  ].forEach(dm=>{
    dm.style.marginLeft = sideWidth;
  })
}

onMounted(async () => {

  let ls = sessionStorage.getItem("lang") || "0";
  let lang = parseInt(ls);

  language.value = Langs[lang].label;
  data.lang = Base.GetLang(I18n);

  expand();

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

const {nodes, activeNodeId} = toRefs(data);

</script>

<style scoped>

.main-header {
  transition: margin-left .3s ease-in-out;
  margin-left: 15vw;
  background-color: #B197FC;
  border-bottom: 1px solid #4b545c;
  z-index: 1034;
  color: #ffffff;
}
.navbar {
  font-size: 1.05rem;
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

