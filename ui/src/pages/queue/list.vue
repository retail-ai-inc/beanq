<template>
    <div class="channel">
      <Pagination :page="page" :total="total" @changePage="changePage"/>

      <div class="accordion" id="ui-accordion">
        <div class="accordion-item" v-if="queues.length === 0">
          Hurrah! We processed all messages.
        </div>
        <div class="accordion-item" v-else v-for="(item, key) in queues" :key="key" style="margin-bottom: 0.9375rem">
          <h2 class="accordion-header">
            <button style="font-weight: bold" class="accordion-button" type="button" data-bs-toggle="collapse" :data-bs-target="setId(key)" aria-expanded="true" :aria-controls="key">
              {{key}}
            </button>
          </h2>
          <div :id="key" class="accordion-collapse collapse show" data-bs-parent="#ui-accordion">
            <div class="accordion-body" style="padding: 0.5rem">
              <table class="table table-striped">
                <thead>
                <tr>
                  <th scope="col">Topic</th>
                  <th scope="col">State</th>
                  <th scope="col">Memory usage</th>
                  <th scope="col">Idle</th>
                  <th scope="col">Action</th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="(d, k) in item" :key="k">
                  <th scope="row">
                    <router-link to="" class="nav-link text-muted" v-on:click="detailQueue(d)">{{ d.topic }}</router-link>
                  </th>
                  <td :class="d.state == 'Run' ? 'text-success-emphasis' : 'text-danger-emphasis'" class="align-middle">{{ d.state }}</td>
                  <td class="align-middle">{{ d.size }}</td>
                  <td class="align-middle">{{ d.idle }}</td>
                  <td>
                    <div class="btn-group" role="group" aria-label="Button group with nested dropdown">
                      <div class="btn-group" role="group">
                        <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
                          Actions
                        </button>
                        <ul class="dropdown-menu">
                          <li><a class="dropdown-item" href="#">Delete</a></li>
                          <li><a class="dropdown-item" href="#">Pause</a></li>
                        </ul>
                      </div>
                    </div>
                  </td>
                </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
      <Pagination :page="page" :total="total" @changePage="changePage"/>
    </div>
</template>
  
  
<script setup>

import { reactive,onMounted,toRefs,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';
import Pagination from "../components/pagination.vue";

let pageSize = 10;
let data = reactive({
  queues:[],
  page:1,
  total:1
})

function getQueue(page,pageSize){
  return request.get("queue/list",{"params":{"page":page,"pageSize":pageSize}});
}

onMounted(async ()=>{
  let queue = await getQueue(data.page,10);
  data.queues = {...queue.data};
})

async function changePage(page){
  let queue = await getQueue(page,10);
  data.queues = {...queue.data.data};
  data.total = Math.ceil(queue.data.total / 10);
  data.page = page;
}

function setId(id){
  return "#"+id;
}

const uRouter = useRouter();
function detailQueue(item){
  uRouter.push("queue/detail/"+item.channel + ":" + item.topic);
}

const {queues,page,total} = toRefs(data);
</script>
  
<style scoped>
.table .text-success-emphasis {
    color: var(--bs-green) !important;
}
.table-striped th{
  font-weight: 400 !important;
}
.table .text-danger-emphasis {
    color: var(--bs-danger) !important;
}
.channel {
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>
  
  