<template>
    <div>
        <table class="table table-striped">
            <thead>
                <tr>
                  <th scope="col">Group</th>
                    <th scope="col">Queue</th>
                    <th scope="col">State</th>
                    <th scope="col">Size</th>
                    <th scope="col">Memory usage</th>
                    <th scope="col">Processed</th>
                    <th scope="col">Failed</th>
                    <th scope="col">Action</th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="(item,key) in schedule" :key="key">
                  <th scope="row">{{item.group}}</th>
                    <th scope="row">{{ item.queue }}</th>
                    <td :class="item.state == 'Run' ? 'text-success-emphasis': 'text-danger-emphasis'">{{ item.state }}</td>
                    <td>{{ item.size }}</td>
                    <td>{{ item.memory }}</td>
                    <td>{{ item.process }}</td>
                    <td>{{ item.fail }}</td>
                    <td>
                      <div class="btn-group" role="group" aria-label="Button group with nested dropdown">
                        <div class="btn-group" role="group">
                          <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
                            Actions
                          </button>
                          <ul class="dropdown-menu">
                            <li><a class="dropdown-item">Delete</a></li>
                            <li><a class="dropdown-item">Retry</a></li>
                            <li><a class="dropdown-item">Archive</a></li>
                          </ul>
                        </div>
                      </div>
                    </td>
                </tr>
            </tbody>

        </table>
      <Pagination :page="page" :total="total" @changePage="changePage"/>
    </div>
</template>
  
  
<script setup>

import { reactive,toRefs,onMounted,onUnmounted } from "vue";
import request  from "request";
import Pagination from "./components/pagination.vue";

const data = reactive({
  page:1,
  total:1,
  schedule:[]
})
function getSchedule(page,pageSize){
  return request.get("schedule",{"params":{"page":page,"pageSize":pageSize}});
}
async function changePage(page){
  let schedule = await getSchedule(page,10);
  data.schedule = {...schedule.data};
  data.total = Math.ceil(schedule.data.total / 10);
  data.page = page;
}
onMounted(async ()=>{
  let schedule = await getSchedule(data.page,10);
  data.schedule = {...schedule.data};
  data.total = Math.ceil(schedule.data.total / 10);
})
const {page,total,schedule} = toRefs(data);
</script>
  
<style scoped>
.table .text-success-emphasis{
    color:var(--bs-green) !important;
}
.table .text-danger-emphasis{
    color:var(--bs-danger) !important;
}
</style>
  
  