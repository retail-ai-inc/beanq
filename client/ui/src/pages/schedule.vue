<template>
    <div>
        <table class="table table-striped">
            <thead>
                <tr>
                    <th scope="col">Queue</th>
                    <th scope="col">State</th>
                    <th scope="col">Size</th>
                    <th scope="col">Memory usage</th>
                    <th scope="col">Processed</th>
                    <th scope="col">Failed</th>
                    <th scope="col">Error rate</th>
                    <th scope="col">Action</th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="(item,key) in schedule" :key="key">
                    <th scope="row">{{ item.queue }}</th>
                    <td :class="item.state == 'Run' ? 'text-success-emphasis': 'text-danger-emphasis'">{{ item.state }}</td>
                    <td>{{ item.size }}</td>
                    <td>{{ item.memory }}</td>
                    <td>{{ item.process }}</td>
                    <td>{{ item.fail }}</td>
                    <td>{{ item.errRate }}</td>
                    <td>...</td>
                </tr>
            </tbody>

        </table>
    </div>
</template>
  
  
<script setup>

import { reactive,onMounted,onUnmounted } from "vue";
import request  from "request";

const schedule = reactive([])
function getQueue(){
  return request.get("schedule");
}
onMounted(async ()=>{
  let data = await getQueue();
  Object.assign(schedule,data.data);

})

</script>
  
<style scoped>
.table .text-success-emphasis{
    color:var(--bs-green) !important;
}
.table .text-danger-emphasis{
    color:var(--bs-danger) !important;
}
</style>
  
  