<template>
    <div>
      <Pagination :page="page" :total="total" @changePage="changePage"/>

        <table class="table table-striped" style="table-layout: fixed">
            <thead>
                <tr>
                    <th scope="col" style="width:3%">Key</th>
                    <th scope="col" style="width:5%">TTL(s)</th>
                    <th scope="col" style="width:10%">AddTime</th>
                    <th scope="col" style="width:5%">RunTime</th>
                    <th scope="col" style="width:6%">Group</th>
                    <th scope="col" style="width:10%">Queue</th>
                    <th scope="col" style="width:35%">Payload</th>
                    <th scope="col" style="width:6%">Action</th>
                </tr>
            </thead>
            <tbody class="table-body">
                <tr v-for="(item, key) in logs" :key="key">
                    <th scope="row">{{ item.key}}</th>
                    <td>{{ item.ttl }}</td>
                    <td>{{item.addTime}}</td>
                    <td>{{item.runTime}}</td>
                    <td>{{ item.group }}</td>
                    <td>{{item.queue}}</td>
                    <td>{{item.payload}}</td>
                    <td>
                      <div class="btn-group" role="group" aria-label="Button group with nested dropdown">
                        <div class="btn-group" role="group">
                          <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
                            Actions
                          </button>
                          <ul class="dropdown-menu">
                            <li><a class="dropdown-item" @click="options('delete',item.key)">Delete</a></li>
                            <li><a class="dropdown-item" @click="options('retry',item.key)">Retry</a></li>
                            <li><a class="dropdown-item" @click="options('archive',item.key)">Archive</a></li>
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
import Pagination from "../components/pagination.vue";

let pageSize = 10;
let data = reactive({
  logs:[],
  page:1,
  total:1
})
// success logs
function getLog(page,pageSize){
  return request.get("log",{"params":{"type":"success","page":page,"pageSize":pageSize}});
}
onMounted(async ()=>{
  let logs = await getLog(data.page,10);
  data.logs = {...logs.data.data};
  data.total = Math.ceil(logs.data.total/pageSize);
})
// click pagination
async function changePage(page){
  let logs = await getLog(page,10);
  data.logs = {...logs.data.data};
  data.total = Math.ceil(logs.data.total / 10);
  data.page = page;

}
async function options(optType,id){
  switch (optType){
    case "delete":
      await request.delete("/log/del", {params: {id: id}}).then(res=>{
        getLog(data.page,10);
      }).catch(err=>{
        console.error(err)
      })
    case "retry":
      await request.post("/log/retry",{id:id},{headers:{"Content-Type":"multipart/form-data"}} ).then(res=>{
        getLog(data.page,10);
      }).catch(err=>{
        console.error(err)
      })
    case "archive":

    default:


  }
}
const {logs,page,total} = toRefs(data);

</script>
  
<style scoped>
.table .table-body th,.table .table-body td{
  vertical-align: middle;
}
.table .text-success-emphasis {
    color: var(--bs-green) !important;
}

.table .text-danger-emphasis {
    color: var(--bs-danger) !important;
}
.table-body tr td{
  word-break:break-all;overflow:hidden;
}
.dropdown-menu .dropdown-item{cursor: pointer}
</style>
  
  