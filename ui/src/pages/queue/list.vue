<template>
    <div class="channel">

      <Spinner v-if="loading"/>
      <div v-else>
        <NoMessage v-if="JSON.stringify(queues) == '{}'"/>
        <div v-else>
          <Pagination v-if="queues.length > 0" :page="page" :total="total" @changePage="changePage"/>
          <div class="accordion" id="ui-accordion">

            <div class="accordion-item" v-for="(item, key) in queues" :key="key" style="margin-bottom: 0.9375rem">
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
                      <th scope="col">Memory usage(byte)
                        <HelpIcon title="The number of bytes required to serialize the object, which can be used to estimate memory usage"/>
                      </th>
                      <th scope="col">Idle(s)
                        <HelpIcon title="Represents the number of seconds since a key was last accessed"/>
                      </th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr v-for="(d, k) in item" :key="k">
                      <th scope="row">
                        <router-link to="" class="nav-link text-muted" v-on:click="detailQueue(d)" style="color:#0d6efd !important;">{{ d.topic }}</router-link>
                      </th>
                      <td :class="d.state == 'Run' ? 'text-success-emphasis' : 'text-danger-emphasis'" class="align-middle">{{ d.state }}</td>
                      <td class="align-middle">{{d.moodType.replace("_stream","")}}</td>
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
        </div>
      </div>

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
import Spinner from "../components/spinner.vue";
import NoMessage from "../components/noMessage.vue";
import HelpIcon from "../components/icons/help_icon.vue";

const [queues,page,pageSize,total,uRouter] = [ref([]),ref(1),ref(100),ref(1),useRouter()];
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const [id,toastRef] = [ref("userToast"),ref(null)];
const loading = ref(false);

const getQueue=(async (pageCur,pageSize)=>{
  loading.value = true;
  try {
    let res = await request.get("queue/list",{"params":{"page":pageCur,"pageSize":pageSize}});
    queues.value = res ?? [];
    page.value = pageCur;
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
  
  