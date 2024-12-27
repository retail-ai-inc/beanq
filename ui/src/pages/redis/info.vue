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
  </div>
</template>
  
  
<script setup>

import { reactive,onMounted,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';

let info = reactive({
  sse:null
});

const useRe = useRouter();

onMounted(async ()=>{

  // bootstrap tooltips
  const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]')
  const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl))

  if(info.sse){
    info.sse.close();
  }
  info.sse = sseApi.Init("redis");
  info.sse.onopen = () => {
    console.log("success")
  }
  info.sse.addEventListener("redis_info",function (res) {
    let body = JSON.parse(res.data);
    if (body.code !== "0000"){
      return
    }
    Object.assign(info,body.data);
  })
  info.sse.onerror = (err)=>{
    //useRe.push("/login");
    console.log(err)
  }

})

onUnmounted(()=>{
  info.sse.close();
})

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
  
  