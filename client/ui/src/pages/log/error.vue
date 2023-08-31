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
                      <div class="btn-group" role="group" aria-label="Button group with nested dropdown">
                        <div class="btn-group" role="group">
                          <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
                            Actions
                          </button>
                          <ul class="dropdown-menu">
                            <li><a type="button" class="dropdown-item" href="#">Delete</a></li>
                            <li><a type="button" class="dropdown-item" href="#">Retry</a></li>
                            <li><a type="button" class="dropdown-item" href="#">Archive</a></li>
                          </ul>
                        </div>
                      </div>
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
  return request.get("log",{"params":{"type":"error","page":0,"pageSize":10}});
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
  
  