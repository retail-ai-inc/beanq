<template>
  <nav class="main-header navbar navbar-expand navbar-danger navbar-dark">
    <div style="width: 1.5rem;height:1.5rem" @click="expand">
        <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-list" viewBox="0 0 16 16">
          <path fill-rule="evenodd" d="M2.5 12a.5.5 0 0 1 .5-.5h10a.5.5 0 0 1 0 1H3a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0 1 .5-.5h10a.5.5 0 0 1 0 1H3a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0 1 .5-.5h10a.5.5 0 0 1 0 1H3a.5.5 0 0 1-.5-.5z"/>
        </svg>
    </div>
    <div class="container-fluid">
      <ul class="navbar-nav">
        <li class="nav-item" v-for="(item,key) in nav" :key="key" :class="item.children.length > 0 ? 'dropdown':''">
            <div v-if="item.children.length > 0 ">
              <a class="nav-link dropdown-toggle" v-if="hasRoles.includes(item.id)" :class="item.tos.indexOf(route) !== -1 ? 'active' : ''" role="button" data-bs-toggle="dropdown" aria-expanded="false">
                {{$t(item.mark)}}
              </a>
              <ul class="dropdown-menu dropdown-menu-color">
                <li v-for="(val,ind) in item.children" :key="ind">
                  <router-link :to="val.to" v-if="hasRoles.includes(val.id)" class="dropdown-item" :class="route === val.to ? 'active' : ''">
                    {{$t(val.mark)}}
                  </router-link>
                </li>
              </ul>
            </div>
            <div v-else>
              <!--nav no sub-->
              <router-link v-if="item.children.length <= 0 && (hasRoles.includes(item.id))" :to="item.to" class="nav-link" :class="route === item.to ? 'active' : ''">
                {{$t(item.mark)}}
              </router-link>
            </div>
        </li>
      </ul>
      <ul class="navbar-nav">
        <li class="nav-item dropdown">
          <div >
            <a class="nav-link dropdown-toggle"  role="button" data-bs-toggle="dropdown" aria-expanded="false">
              {{language}}
            </a>
            <ul class="dropdown-menu dropdown-menu-color">
              <li v-for="(val,ind) in hlang" :key="ind">
                <a  class="dropdown-item" :class="route === val.to ? 'active' : ''" @click="action(val)">
                  {{val.label}}
                </a>
              </li>
            </ul>
          </div>
        </li>
        <li class="nav-item">
          <a class="nav-link" @click="jump('','Logout')">{{$t("logOut")}}</a>
        </li>
      </ul>
    </div>
  </nav>

</template>

<script setup>

import {useRoute, useRouter} from 'vueRouter';
import {ref, toRefs, onMounted, watch, reactive,defineProps,defineEmits} from "vue";
import i18n from "i18n";
const props = defineProps({
  nav:{},
  hlang:{},
})

const data = reactive({
  nodes: [],
  activeNodeId: "",
  isSide:false,
})
const [lang,nav,langTag] = [ref(props.hlang),ref(props.nav),ref("en")];

const [route,uroute,urouter,language] = [ref("/admin/home"),useRoute(),useRouter(),ref("English")];

const emits = defineEmits(['action']);
const action = function (obj){

  i18n.global.locale.value = obj.flag;
  emits("action",obj);
  language.value = obj.label;
  sessionStorage.setItem("i18n",obj.flag);
}

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

const hasRoles = ref([]);
onMounted(async () => {

  langTag.value = sessionStorage.getItem("i18n") || "en";
  language.value = _.find(props.hlang,(v)=>{
    return v.flag === langTag.value;
  })?.label;

  i18n.global.locale.value = langTag.value;

  let roles = JSON.parse(sessionStorage.getItem("roles"));
  if(_.isEmpty(roles)){
    let navs = roleApi.TileTree(nav.value);
    for(let i = 0;i<navs.length;i++){
      hasRoles.value.push(navs[i].id);
    }
  }else{
    hasRoles.value = roles;
  }
  expand();

  route.value = uroute.path;
})

watch(() => uroute.path, (newVal, oldVal) => {
  route.value = newVal;

  document.querySelectorAll('.dropdown-toggle').forEach((toggle) => {
    toggle.setAttribute('aria-expanded', 'false');
  });
  document.querySelectorAll('.dropdown-menu').forEach((menu) => {
    menu.classList.remove('show');
  });
})

function jump(uri,flag){
  if(flag === "Logout"){
    sessionStorage.clear();
    urouter.replace("/login");
  }else{
    urouter.push(uri);
  }
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

