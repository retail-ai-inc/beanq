<template>
    <div class="container-fluid" style="padding: 1.5rem">

      <table class="table table-striped" style="table-layout: fixed">
        <thead>
        <tr>
          <th scope="col" style="width:8%">Id</th>
          <th scope="col" style="width:8%">Topic</th>
          <th scope="col" style="width:10%">RegisteredAt</th>
          <th scope="col" style="width:8%">Priority</th>
          <th scope="col" style="width:6%">Channel</th>
          <th scope="col" style="width:10%">Topic</th>
          <th scope="col" style="width:35%">Payload</th>
        </tr>
        </thead>
        <tbody class="table-body">
        <tr v-if="queueDetail.length === 0">
          <th scope="row" colspan="7" style="text-align: center">
            Hurrah! We processed all messages.
          </th>
        </tr>
        <tr v-else v-for="(item, key) in queueDetail" :key="key">
          <th scope="row">
            {{item.ID}}
          </th>
          <td>{{ item.Values.topic }}</td>
          <td>{{item.Values.addTime}}</td>
          <td>{{item.Values.priority}}</td>
          <td>{{ item.Values.channel }}</td>
          <td>{{item.Values.topic}}</td>
          <td>{{item.Values.payload}}</td>
        </tr>
        </tbody>
      </table>
      <LoginModal :id="loginId" ref="loginModal"/>
    </div>

</template>
<script setup>

import {ref,reactive,toRefs,onMounted,onUnmounted} from "vue";
import { useRoute,useRouter } from 'vueRouter';
import cfg  from "config";
import LoginModal from "../components/loginModal.vue";

let data = reactive({
  queueDetail:[],
  sseEvent:null,
});
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];

const uRoute = useRoute();
let id = uRoute.params.id;

function initEventSource(){

  let apiUrl = `${cfg.sseUrl}queue/detail?id=${id}&token=${Storage.GetItem("token")}`;
  if (data.sseEvent){
    data.sseEvent.close();
  }
  data.sseEvent = sseApi.Init(apiUrl);
  data.sseEvent.onopen = () =>{
    console.log("handshake success");
  }
  data.sseEvent.onerror = (err)=>{
    console.log(err.error);
    data.sseEvent.close();
    setTimeout(initEventSource,3000);
  }
  data.sseEvent.addEventListener("queue_detail", async function(res){
    let body =  JSON.parse(res.data);

    if (body.code === "1004"){
      loginModal.value.error(new Error(body.msg));
      data.sseEvent.close();
      return
    }

    data.eventLogs = body.data.data;
    data.page =  body.data.cursor;
    data.total = body.data.total;
  })
}


onMounted( async ()=>{
  initEventSource();
})

onUnmounted(()=>{
  data.sseEvent.close();
})
const {queueDetail} = toRefs(data);
</script>