<template>
    <div class="channel">
      <Pagination v-if="queues.length > 0" :page="page" :total="total" @changePage="changePage"/>
      <div class="accordion" id="ui-accordion">
        <div class="accordion-item" v-if="JSON.stringify(queues) == '{}'" style="border: none;text-align: center;padding: 0.9375rem 0">
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
                  <th scope="col">Mood Type</th>
                  <th scope="col">Memory usage(byte)</th>
                  <th scope="col">Idle(s)</th>
                </tr>
                </thead>
                <tbody>
                <tr v-for="(d, k) in item" :key="k">
                  <th scope="row">
                    <router-link to="" class="nav-link text-muted" v-on:click="detailQueue(d)">{{ d.topic }}</router-link>
                  </th>
                  <td :class="d.state == 'Run' ? 'text-success-emphasis' : 'text-danger-emphasis'" class="align-middle">{{ d.state }}</td>
                  <td class="align-middle">{{d.moodType}}</td>
                  <td class="align-middle">{{ d.size }}</td>
                  <td class="align-middle">{{ d.idle }}</td>
                </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
      <Pagination v-if="queues.length > 0" :page="page" :total="total" @changePage="changePage"/>
      <LoginModal :id="loginId" ref="loginModal"/>
      <Btoast :id="id" ref="toastRef">
      </Btoast>

    </div>
</template>

<script setup>
import { ref,onMounted } from "vue";
import { useRouter } from 'vueRouter';
import Pagination from "../components/pagination.vue";
import LoginModal from "../components/loginModal.vue";
import Btoast from "../components/btoast.vue";

const [queues,page,pageSize,total,uRouter] = [ref([]),ref(1),ref(10),ref(1),useRouter()];
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const [id,toastRef] = [ref("userToast"),ref(null)];

const getQueue=(async (pageCur,pageSize)=>{
  try {
    let res = await request.get("queue/list",{"params":{"page":pageCur,"pageSize":pageSize}});
    toastRef.value.show("success");
    queues.value = res;
    page.value = pageCur;
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e);
  }
})

onMounted( ()=>{
  getQueue(page.value,10);
})

async function changePage(page){
  try {
    await getQueue(page,10)
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e);
  }
}

function setId(id){
  return "#"+id;
}

function detailQueue(item){
  uRouter.push("queue/detail/"+item.channel + ":" + item.topic);
}

</script>
  
<style scoped>

.table{
  .text-success-emphasis {
    color: var(--bs-green) !important;
  }
  .text-danger-emphasis {
    color: var(--bs-danger) !important;
  }
}

.table-striped th{
  font-weight: 400 !important;
}

.channel {
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>
  
  