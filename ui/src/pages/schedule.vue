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
                    <a class="btn btn-danger icon-button" href="javascript:;" role="button" title="Delete" @click="deleteModal(item)">
                      <DeleteIcon />
                    </a>
                  </td>
                </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
      <Pagination :page="page" :total="total" @changePage="changePage"/>

      <Btoast :id="id" ref="toastRef">
      </Btoast>

    </div>
</template>
  
  
<script setup>

import { ref,onMounted } from "vue";
import Pagination from "./components/pagination.vue";
import DeleteIcon from "./components/icons/delete_icon.vue";
import Btoast from "./components/btoast.vue";

const [
  page,
  total,
  schedules,
  id,
  toastRef
] = [ref(1),ref(1),ref([]),ref("liveToast"),ref(null)];

function deleteModal(item){}

async function changePage(page,cursor){
  try {
    let scheduleData = await scheduleApi.GetSchedule(page,10);
    const {code,data,msg} = scheduleData;
    if(code !== "0000"){
      return
    }
    schedules.value = data;
    total.value = Math.ceil(data.total / 10);
    page.value = page;
  }catch (e) {
    toastRef.value.show(e);
  }

}

onMounted(async ()=>{
  try{
    let scheduleData = await scheduleApi.GetSchedule(page.value,10);
    schedules.value = scheduleData.data;
    total.value = Math.ceil(scheduleData.data.total / 10);
  }catch (e) {
    toastRef.value.show(e);
  }
})

function setScheduleId(id){
  return "#"+id;
}

</script>
  
<style scoped>

.table{
  .text-success-emphasis{
    color:var(--bs-green) !important;
  }
  .text-danger-emphasis{
    color:var(--bs-danger) !important;
  }
}

.schedule{
  transition: opacity 0.5s ease;
  opacity: 1;
}
.icon-button{
  width: 2.2rem;height:2.2rem;padding:0.2rem 0.5rem 0.5rem;margin-right: 0.2rem;
}
</style>
  
  