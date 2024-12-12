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

    </div>

</template>
<script setup>

import {reactive,toRefs,onMounted,onUnmounted} from "vue";
import { useRoute,useRouter } from 'vueRouter';
import cfg  from "config";

let data = reactive({
  queueDetail:[]
});

const uRoute = useRoute();
let id = uRoute.params.id;

let sseUrl = `${cfg.sseUrl}queue/detail?id=${id}&token=${sessionStorage.getItem("token")}`;
const sse = new EventSource(sseUrl);

onMounted( async ()=>{

  sse.onopen = ()=>{
    console.log("success")
  }
  sse.addEventListener("queue_detail",function (res) {
    let body = JSON.parse(res.data);
    if (body.code !== "0000"){
      return
    }
    data.queueDetail = body.data;
  })
  sse.onerror = (err)=>{
    useRe.push("/login");
    console.log(err)
  }
})

onUnmounted(()=>{
  sse.close();
})
const {queueDetail} = toRefs(data);
</script>