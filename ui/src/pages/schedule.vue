<template>
    <div class="schedule">
      <Spinner v-if="loading"/>
      <div v-else>

        <NoMessage v-if="JSON.stringify(schedules) == '{}'" />
        <div v-else>
          <Pagination :page="page" :total="total" @changePage="changePage"/>
          <div class="accordion" id="schedule-ui-accordion">

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
                    </tr>
                    </thead>
                    <tbody>
                    <tr v-for="(d, k) in item" :key="k">
                      <th scope="row">{{ d.topic }}</th>
                      <td :class="d.state == 'Run' ? 'text-success-emphasis' : 'text-danger-emphasis'">{{ d.state }}</td>
                      <td>{{ d.size }}</td>
                      <td>{{ d.memory }}</td>
                      <td>{{ d.process }}</td>
                    </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
          <Pagination :page="page" :total="total" @changePage="changePage"/>
        </div>
      </div>

      <Btoast :id="id" ref="toastRef">
      </Btoast>
      <LoginModal :id="loginId" ref="loginModal"/>
    </div>
</template>
  
  
<script setup>

import { ref,onMounted } from "vue";
import Pagination from "./components/pagination.vue";
import DeleteIcon from "./components/icons/delete_icon.vue";
import Btoast from "./components/btoast.vue";
import Log from "./log.vue";
import LoginModal from "./components/loginModal.vue";
import Spinner from "./components/spinner.vue";
import NoMessage from "./components/noMessage.vue";

const [
  page,
  total,
  schedules,
  id,
  toastRef
] = [ref(1),ref(1),ref([]),ref("liveToast"),ref(null)];
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const loading = ref(false);

function deleteModal(item){}

const changePage = ((page,cursor)=>{
  getSchedule(page);
})

const getSchedule = (async (pageCurr)=>{
  loading.value = true;
  try{
    let data = await scheduleApi.GetSchedule(pageCurr,10);
    schedules.value = data;
    total.value = Math.ceil(data.length / 10);
    setTimeout(()=>{
      loading.value = false;
    },800)
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e);
  }
})

onMounted(()=>{
  getSchedule(page.value);
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
  
  