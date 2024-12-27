<template>
    <div>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
        <table class="table table-striped">
            <thead>
                <tr>
                  <th scope="col" style="width:8%">Id</th>
                  <th scope="col" style="width:8%">TTL(s)</th>
                  <th scope="col" style="width:10%">RegisteredAt</th>
                  <th scope="col" style="width:8%">ProcessingTime</th>
                  <th scope="col" style="width:6%">Channel</th>
                  <th scope="col" style="width:10%">Topic</th>
                  <th scope="col" style="width:35%">Payload</th>
                  <th scope="col" style="width:6%">Action</th>
                </tr>
            </thead>
            <tbody>
            <tr v-for="(item, key) in logs" :key="key">
              <th scope="row">
                <router-link to="" class="nav-link text-muted" v-on:click="detail(item)">{{item.id}}</router-link>
              </th>
              <td>{{ item.expireTime }}</td>
              <td>{{item.addTime}}</td>
              <td>{{item.runTime}}</td>
              <td>{{ item.channel }}</td>
              <td>{{item.topic}}</td>
              <td>
                <div style="height: 3rem;overflow: hidden;white-space: nowrap;width: 36rem;text-overflow: ellipsis;">
                {{item.payload}}
                </div>
              </td>
              <td>
                <div class="btn-group" role="group" aria-label="Button group with nested dropdown">
                  <div class="btn-group" role="group">
                    <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
                      Actions
                    </button>
                    <ul class="dropdown-menu">
                      <li><a class="dropdown-item" @click="options('delete',item)">Delete</a></li>
                      <li><a class="dropdown-item" @click="options('retry',item)">Retry</a></li>
                      <li><a class="dropdown-item" @click="options('archive',item.id)">Archive</a></li>
                    </ul>
                  </div>
                </div>
              </td>
            </tr>
            </tbody>
        </table>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
    </div>
</template>
  
  
<script setup>

import { useRouter } from 'vueRouter';
import { reactive,toRefs,onMounted,onUnmounted } from "vue";
import Pagination from "../components/pagination.vue";

let pageSize = 10;
let data = reactive({
  logs:[],
  page:1,
  total:1,
  cursor:0
})

function getErrLog(page,pageSize,cursor){
  return request.get("logs",{"params":{"type":"error","page":page,"pageSize":pageSize,"cursor":cursor}});
}

const uRouter = useRouter();
function detail(item){
  uRouter.push("detail/"+item.id+"/fail");
}

onMounted(async ()=>{

  let logs = await getErrLog(data.page,10,data.cursor);
  data.logs = {...logs.data.data};
  data.total = Math.ceil(logs.data.total/pageSize);
  data.cursor = logs.data.cursor;

})

// click pagination
async function changePage(page,cursor){
  let logs = await getErrLog(page,10,cursor);

  data.logs = {...logs.data.data};
  data.total = Math.ceil(logs.data.total / 10);
  data.page = page;
  data.cursor = logs.data.cursor;

}

async function options(optType,param){

  switch (optType){
    case "delete":
      await request.delete("/log",  {"params":{score: param.Score,msgType:"fail"}}).then(res=>{

        let logs = getErrLog(data.page,10,0);

        data.logs = {...logs.data.data};
        data.total = Math.ceil(logs.data.total / 10);
        data.page = page;
        data.cursor = logs.data.cursor;

      }).catch(err=>{
        console.error(err)
      })
          break;
    case "retry":
      await request.post("/log",{id:param.id,msgType:"fail"},{headers:{"Content-Type":"multipart/form-data"}} ).then(res=>{
        getErrLog(data.page,10,data.cursor);
      }).catch(err=>{
        console.error(err)
      })
          break;
    case "archive":

    default:


  }
}

const {logs,page,total,cursor} = toRefs(data);
</script>
  
<style scoped>
.table .text-success-emphasis {
    color: var(--bs-green) !important;
}

.table .text-danger-emphasis {
    color: var(--bs-danger) !important;
}
</style>
  
  