<template>

  <div class="home">
    <div class="chart-container">
      <div class="chart-h">
        <v-chart class="chart" :option="queuedMessagesOption"/>
      </div>
<!--      <div class="chart-h">-->
<!--        <v-chart class="chart" :option="refererOption"/>-->
<!--      </div>-->
      <div class="chart-h">
        <v-chart class="chart" :option="messageRatesOption"/>
      </div>
<!--      <div class="chart-h">-->
<!--        <v-chart class="chart" :option="barOption"/>-->
<!--      </div>-->
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
const useR = useRouter();
onMounted(async () => {

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
  grid-template-columns: repeat(2, 1fr); /* 两列，每列宽度相等 */
  gap: 10px; /* 可选：设置元素之间的间距 */
}

.chart-h {
  height: 280px;
}

.chart {
  background-color: #ffffff; /* 示例背景色，可以根据需要更改 */
  /*border: 1px solid #ccc; !* 示例边框，可以根据需要更改 *!*/
  box-sizing: border-box; /* 包括边框和内边距在内的宽度和高度计算 */
}


</style>