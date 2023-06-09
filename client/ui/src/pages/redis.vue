<template>
  <div class="container-fluid">
    <div class="card">
      <div class="card-header">
        Cpu
      </div>
      <ul class="list-group list-group-flush">
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_sys</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_cpu_sys">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_sys_children</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_cpu_sys_children">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_user</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_cpu_user">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_cpu_user_children</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_cpu_user_children">
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
            <label class="col-sm-2 col-form-label">mem_fragmentation_ratio</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.mem_fragmentation_ratio">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_dataset_perc</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_dataset_perc">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_human</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_human">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_peak</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_peak">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_peak_human</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_peak_human">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_peak_perc</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_peak_perc">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_rss</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_rss">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">used_memory_rss_human</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.used_memory_rss_human">
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
            <label class="col-sm-2 col-form-label">redis_build_id</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.redis_build_id">
            </div>
          </div>
        </li>
        <li class="list-group-item">
          <div class="mb-3 row">
            <label class="col-sm-2 col-form-label">redis_version</label>
            <div class="col-sm-10">
              <input type="text" readonly class="form-control-plaintext" :value="info.redis_version">
            </div>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>
  
  
<script setup>

import { reactive,onMounted,onUnmounted } from "vue";
import request  from "request";

let info = reactive({});

 function getData(){
  return request.get("redis");
}
onMounted(async ()=>{
    let data = await getData();
    Object.assign(info,data.data);

})

let loopData = setInterval(  async function (){
  let data = await getData();
  Object.assign(info,data.data);
},10000);

onUnmounted(()=>{
  clearInterval(loopData);
})

</script>
  
<style scoped>
.example {
    color: v-bind('color');
}
.card{
  margin-bottom: 10px;
}
.card .card-header{
  font-weight: bold;
}
.mb-3{
  margin-bottom: 0 !important;
}
</style>
  
  