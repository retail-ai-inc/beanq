<template>
    <div>
        <table class="table table-striped">
            <thead>
                <tr>
                    <th scope="col">Key</th>
                    <th scope="col">TTL(s)</th>
                    <th scope="col">AddTime</th>
                  <th scope="col">RunTime</th>
                  <th scope="col">Group</th>
                  <th scope="col">Queue</th>
                    <th scope="col">Payload</th>
                    <th scope="col">Action</th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="(item, key) in logs" :key="key">
                    <th scope="row">{{ item.key }}</th>
                    <td>{{ item.ttl }}</td>
                  <td>{{item.addTime}}</td>
                  <td>{{item.runTime}}</td>
                    <td>{{ item.group }}</td>
                  <td>{{item.queue}}</td>
                    <td>{{ item.payload }}</td>
                    <td>
                        <button type="button" class="btn btn-danger btn-sm" style="font-size: .5rem;margin:0 .5rem">Delete</button>
                        <button type="button" class="btn btn-success btn-sm" style="font-size: .5rem;margin:0 .5rem">Retry</button>
                      <button type="button" class="btn btn-info btn-sm" style="font-size: .5rem;margin:0 .5rem">Archive</button>
                    </td>
                </tr>
            </tbody>
        </table>
    </div>
</template>
  
  
<script setup>

import { reactive,onMounted,onUnmounted } from "vue";
import request  from "request";

const logs = reactive([])
function getLog(){
  return request.get("log");
}
onMounted(async ()=>{
  let data = await getLog();
  Object.assign(logs,data.data);

})
</script>
  
<style scoped>
.table .text-success-emphasis {
    color: var(--bs-green) !important;
}

.table .text-danger-emphasis {
    color: var(--bs-danger) !important;
}
</style>
  
  