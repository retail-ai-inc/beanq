<template>
  <div class="redis-monitor container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of Redis Monitor Info</h5>
      </div>
    </div>
    <ul class="list-group list-group-flush" id="monitor" style="height: 500px;overflow-y: scroll;background-color: #333">
      <li v-for="(v,k) in commands " :key="k" class="list-group-item" style="background-color: #333;color:#f5f5f5">{{v}}</li>
    </ul>
    <LoginModal :id="loginId" ref="loginModal"/>
  </div>
</template>
  
  
<script setup>

import { ref,reactive,onMounted,onUnmounted,toRefs } from "vue";
import { useRouter } from 'vueRouter';
import LoginModal from "../components/loginModal.vue";

let info = reactive({
  sseMonitor:null,
  commands:[]
});

const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];

const useRe = useRouter();

function monitorSSEConnect(){
  if(info.sseMonitor){
    info.sseMonitor.close();
  }
  info.sseMonitor = sseApi.Init("redis/monitor");
  info.sseMonitor.onopen = () => {
    console.log("success")
  }
  info.sseMonitor.addEventListener("redis_monitor",function (res) {
    let body = JSON.parse(res.data);
    if (body.code === "1004"){
      loginModal.value.error(new Error(body.msg));
      info.sseMonitor.close();
      return
    }

    info.commands.push(body.data);
    let div = document.getElementById("monitor");
    div.scrollTop = div.scrollHeight;
  })
  info.sseMonitor.onerror = (err)=>{
    console.log(err);
    info.sseMonitor.close();
    setTimeout(monitorSSEConnect,3000);
  }
}

onMounted(async ()=>{

  monitorSSEConnect();

})

onUnmounted(()=>{
  info.sseMonitor.close();
  info.commands = [];
})
const {commands} = toRefs(info);
</script>
  
<style scoped>
.redis-monitor{
  transition: opacity 0.5s ease;
  opacity: 1;
}
.card{
  margin-bottom: 0.625rem;
}
.card .card-header{
  font-weight: bold;
}
.mb-3{
  margin-bottom: 0 !important;
}
.list-group .col-form-label{
  text-align: right;
}
</style>
  
  