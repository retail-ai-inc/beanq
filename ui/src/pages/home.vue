<template>

  <div class="home">
    <div class="chart-container">
      <div class="chart-h">
        <v-chart class="chart" ref="line1"  :option="queuedMessagesOption"/>
      </div>
      <div class="chart-h">
        <v-chart class="chart" ref="line2" :option="messageRatesOption"/>
      </div>
    </div>

    <div class="container-fluid" style="margin-bottom: 40px">
      <Dashboard :queue_total="queue_total"
                 :num_cpu="num_cpu"
                 :fail_count="fail_count"
                 :success_count="success_count"
                 :db_size="db_size"/>
    </div>
  </div>
</template>

<script setup>
import {ref, reactive, onMounted,onUnmounted, toRefs,} from "vue";
import { useRouter } from 'vueRouter';
import Dashboard from "./components/dashboard.vue";

let data = reactive({
  "queue_total": 0,
  "db_size": 0,
  "num_cpu": 0,
  "fail_count": 0,
  "success_count": 0,
  "queuedMessagesOption":{},
  "messageRatesOption":{},
  "nodeId":"",
  sse:null
})

const line1 = ref(null);
const line2 = ref(null);

function resize(){
  [line1.value,line2.value].forEach(chart=>{
    chart?.resize();
  })
}

const useR = useRouter();
onMounted(async () => {

  window.addEventListener("resize",resize)

  if(data.sse){
    data.sse.close();
  }
  data.sse = sseApi.Init("dashboard");
  data.sse.onopen = () => {
    console.log("success")
  }
  data.sse.addEventListener("dashboard",function (res) {
    let result = JSON.parse(res.data);
    if (result.code !== "0000"){
      return
    }
    Object.assign(data, result.data);

    sessionStorage.setItem("nodeId",data.nodeId);

    data.queuedMessagesOption = dashboardApi.QueueLine(result.data.queues);
    data.messageRatesOption = dashboardApi.MessageRateLine(result.data.queues);
  })
  data.sse.onerror = (err)=>{
    useR.push("/login");
    console.log(err)
  }

})
onUnmounted(()=>{
  data.sse.close();
  window.removeEventListener("resize",resize)
})

const {
  queue_total,
  db_size,
  num_cpu,
  fail_count,
  success_count,
  queuedMessagesOption,
  messageRatesOption,
} = toRefs(data);
</script>
<style scoped>
.home {
  transition: opacity 0.5s ease;
  opacity: 1;
}


.mt-1 {
  margin-top: 1rem;
}

.chart-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}

.chart-h {
  height: 280px;
}

.chart {
  background-color: #ffffff;
  /*border: 1px solid #ccc; */
  box-sizing: border-box;
}


</style>