<template>
    <div class="schedule">

      <div class="accordion" id="schedule-ui-accordion">
        <div class="accordion-item" v-if="JSON.stringify(schedules) == '{}'" style="border: none;text-align: center;padding: 0.9375rem 0">
          Hurrah! We processed all messages.
        </div>
        <div class="accordion-item" v-else v-for="(item, key) in schedules" :key="key" style="margin-bottom: 0.9375rem">
          <h2 class="accordion-header">
            <button style="font-weight: bold" class="accordion-button" type="button" data-bs-toggle="collapse" :data-bs-target="setScheduleId(key)" aria-expanded="true" :aria-controls="key">
              {{key}}
            </button>
          </h2>
          <div :id="key" class="accordion-collapse collapse show" data-bs-parent="#schedule-ui-accordion">
            <div class="accordion-body" style="padding: 0.5rem">
              <table class="table table-striped">
                <thead>
                <tr>
                  <th scope="col">Topic</th>
                  <th scope="col">State</th>
                  <th scope="col">Size</th>
                  <th scope="col">Memory usage</th>
                  <th scope="col">Processed</th>
                  <th scope="col">Action</th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="(d, k) in item" :key="k">
                  <th scope="row">{{ d.topic }}</th>
                  <td :class="d.state == 'Run' ? 'text-success-emphasis' : 'text-danger-emphasis'">{{ d.state }}</td>
                  <td>{{ d.size }}</td>
                  <td>{{ d.memory }}</td>
                  <td>{{ d.process }}</td>
                  <td>
                    <div class="btn-group-sm" role="group" aria-label="Button group with nested dropdown">
                        <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
                          Actions
                        </button>
                        <ul class="dropdown-menu">
                          <li><a class="dropdown-item" href="#">Delete</a></li>
                          <li><a class="dropdown-item" href="#">Pause</a></li>
                        </ul>
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

import { reactive,toRefs,onMounted,onUnmounted } from "vue";
import Pagination from "./components/pagination.vue";

const data = reactive({
  page:1,
  total:1,
  schedules:[]
})

async function changePage(page){
  let scheduleData = await scheduleApi.GetSchedule(page,10);
  data.schedules = {...scheduleData.data};
  data.total = Math.ceil(scheduleData.data.total / 10);
  data.page = page;
}
onMounted(async ()=>{
  let scheduleData = await scheduleApi.GetSchedule(data.page,10);
  data.schedules = {...scheduleData.data};
  data.total = Math.ceil(scheduleData.data.total / 10);
})
function setScheduleId(id){
  return "#"+id;
}
const {page,total,schedules} = toRefs(data);
</script>
  
<style scoped>
.table .text-success-emphasis{
    color:var(--bs-green) !important;
}
.table .text-danger-emphasis{
    color:var(--bs-danger) !important;
}
.schedule{
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>
  
  