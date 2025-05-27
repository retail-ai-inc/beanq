<template>
  <div>
    <GoBackButton/>

    <div class="container-fluid" style="padding: 1.5rem;border-radius: 0.25rem">
      <div class="row" v-for="(item,key) in detail" style="min-height: 2.5rem">
        <div class="col-1" style="font-weight: bold">
          {{key}}
        </div>
        <div class="col mark" style="white-space: pre-wrap;">
          <pre v-if="key === 'Payload'" class="payload-pre"><code class="payload-code">{{ JSON.stringify(JSON.parse(item), null, 2)}}</code></pre>
          <pre v-else><code>{{item}}</code></pre>
        </div>
      </div>
    </div>
    <GoBackButton />

    <Btoast :id="id" ref="toastRef" />

    <LoginModal :id="noticeId" ref="loginModal"/>
  </div>
</template>
<script setup>

import {ref,onMounted} from "vue";
import { useRoute } from 'vueRouter';
import GoBackButton from "../../components/goBackButton.vue";
import Btoast from "../../components/btoast.vue";
import LoginModal from "../../components/loginModal.vue";


const [id,toastRef] = [ref("userToast"),ref(null)];
const [noticeId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
let detail = ref({});

async function getDetail(paramid){
  try {
    let res = await request.get("/event_log/detail",{"params":{"id":paramid}});
    let {_id,id,addTime,channel,executeTime,logType,maxLen,moodType,payload,topic,priority,retry,timeToRun,status} = res;
    detail.value = {
      "Object Id":_id,
      "Message Id":id,
      "Max Length":maxLen,
      "Log Type":logType,
      "Mood Type":moodType,
      "Channel":channel,
      "Topic":topic,
      "Payload":payload,
      "Add Time":addTime,
      "Execute Time":executeTime,
      "Priority":priority,
      "Retry":retry,
      "Time To Run":timeToRun,
      "Status":status
    };
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
}

const uRoute = useRoute();

onMounted( ()=>{
   getDetail(uRoute.params.id)
})


</script>
<style scoped>
.mark{
  .payload-pre{
    background-color: #f5f5f5;
    padding: 10px;
    border-radius: 5px;
    border: 1px solid #ddd;
    overflow: auto;
  }
  .payload-code{
    color: #d63384;
  }
}
</style>