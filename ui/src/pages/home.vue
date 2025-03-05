<template>

  <div class="home" ref="homeEle">
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

    <div v-for="[index, item] in Object.entries(pods)" :key="index" style="margin-bottom: 2rem;">
      <div style="font-weight: bold">{{index}}</div>
      <table class="table">
        <thead>
        <tr>
          <th scope="col">Cpu Total</th>
          <th scope="col">Cpu Percent</th>
          <th scope="col">Memory Total</th>
          <th scope="col">Memory Percent</th>
          <th scope="col">Memory Used</th>
        </tr>
        </thead>
        <tbody>
        <tr>
          <td>{{item.cpuCount}}</td>
          <td>{{item.cpuPercent}}(%)</td>
          <td>{{item.memoryTotal}}(GB)</td>
          <td>{{item.memoryPercent}}(%)</td>
          <td>{{item.memoryUsed}}(MB)</td>
        </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import {ref, onMounted,onUnmounted} from "vue";
import { useRouter } from 'vueRouter';
import Dashboard from "./components/dashboard.vue";

const [line1,line2,useR,homeEle] = [ref(null),ref(null),useRouter(),ref(null)];

let [
    queue_total,
    db_size,
    num_cpu,
    fail_count,
    success_count,
    queuedMessagesOption,
    messageRatesOption,
    nodeId,
    sse,
    resizeObserver,
    pods
  ] = [ref(0),ref(0),ref(0),ref(0),ref(0),ref({}),ref({}),ref(""),ref(null),null,ref({})];


function resize(){
  [line1.value,line2.value].forEach(chart=>{
    chart?.resize();
  })
}

function sseConnect(){
  if(sse.value){
    sse.value.close();
  }
  sse.value = sseApi.Init("dashboard");
  sse.value.onopen = () => {
    console.log("connect success")
  }
  sse.value.addEventListener("dashboard",function (res) {
    //let result = JSON.parse(res.data);
    const {code,msg,data} = JSON.parse(res.data);
    if (code !== "0000"){
      return
    }

    queue_total.value = data.queue_total;
    db_size.value = data.db_size;
    num_cpu.value = data.num_cpu;
    fail_count.value = data.fail_count;
    success_count.value = data.success_count;

    for(let key in data?.pods){
      data.pods[key] = JSON.parse(data.pods[key]);
    }
    pods.value = data.pods;

    queuedMessagesOption.value = dashboardApi.QueueLine(data.queues);
    messageRatesOption.value = dashboardApi.MessageRateLine(data.queues);
  })
  sse.value.onerror = (err)=>{
    sse.value.close();
    setTimeout(sseConnect,3000);
  }
}

onMounted( () => {

  const parentEle = homeEle.value.parentElement;
  resizeObserver = new ResizeObserver((entries)=>{
    resize();
  })
  Base.Debounce( resizeObserver.observe(parentEle),300) ;

  sseConnect();

})
onUnmounted(()=>{

  if(sse.value){
    sse.value.close();
  }
  if(resizeObserver){
    resizeObserver.disconnect();
  }
})

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