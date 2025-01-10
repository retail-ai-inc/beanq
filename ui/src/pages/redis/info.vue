<template>
  <div class="redis container-fluid">
    <div class="card">
      <div class="card-header">
        Cpu
      </div>
      <ul class="list-group list-group-flush">
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_sys:
            </label>
            <div class="col-sm-10">
              <div class="form-control-plaintext" >{{info.used_cpu_sys}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_sys_children:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_cpu_sys_children}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_user:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_cpu_user}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_user_children:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_cpu_user_children}}</div>
            </div>
          </div>
        </li>
      </ul>
    </div>
    <div class="card">
      <div class="card-header">
        Memory
      </div>
      <ul class="list-group list-group-flush">
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">mem_fragmentation_ratio(<b>%</b>):</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.mem_fragmentation_ratio}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory(<b>Byte</b>):</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_dataset_perc:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_dataset_perc}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_human:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_human}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_peak(<b>Byte</b>):</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_peak}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_peak_human:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_peak_human}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_peak_perc:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_peak_perc}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_rss(<b>Byte</b>):</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_rss}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_rss_human:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_rss_human}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_lua(<b>Byte</b>):</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_lua}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_lua_human:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.used_memory_lua_human}}</div>
            </div>
          </div>
        </li>
      </ul>
    </div>
    <div class="card">
      <div class="card-header">
        Server
      </div>
      <ul class="list-group list-group-flush">
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">redis_build_id:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.redis_build_id}}</div>
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">redis_version:</label>
            <div class="col-sm-10">
              <div class="form-control-plaintext">{{info.redis_version}}</div>
            </div>
          </div>
        </li>
      </ul>
    </div>

    <div class="container-fluid text-center">
      <div class="row justify-content-between">
        <div class="col-4">
          <Command :commands="commands"/>
          <KeySpace :keyspace="keyspace" class="mt-1"/>
        </div>
        <div class="col-4">
          <Client :clients="clients"/>
          <Memory :memory="memory" class="mt-1"/>
        </div>
        <div class="col-4">
          <Stats :stats="stats"/>
        </div>
      </div>
    </div>
  </div>
</template>
  
  
<script setup>

import { reactive,onMounted,onUnmounted,toRefs } from "vue";
import { useRouter } from 'vueRouter';
import Command from "../components/command.vue";
import Client from "../components/client.vue";
import Memory from "../components/memory.vue";
import KeySpace from "../components/keySpace.vue";
import Stats from "../components/stats.vue";

let data = reactive({
  "info":{},
  "commands": [],
  "keyspace": [],
  "clients": {},
  "stats": {},
  "memory": {},
  sse:null
});

const useRe = useRouter();

onMounted(async ()=>{

  // bootstrap tooltips
  const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]')
  const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl))

  if(data.sse){
    data.sse.close();
  }
  data.sse = sseApi.Init("redis");
  data.sse.onopen = () => {
    console.log("success")
  }
  data.sse.addEventListener("redis_info",function (res) {
    let body = JSON.parse(res.data);
    if (body.code !== "0000"){
      return
    }
    Object.assign(data,body.data);
  })
  data.sse.onerror = (err)=>{
    //useRe.push("/login");
    console.log(err)
  }

})

onUnmounted(()=>{
  data.sse.close();
})

const{commands,clients, stats,keyspace,memory,info} = toRefs(data);
</script>
  
<style scoped>
.redis{
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
  
  