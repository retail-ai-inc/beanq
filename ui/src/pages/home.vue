<template>

  <div class="home" ref="homeEle">
    <div class="row justify-content-end">
      <div class="col-1">
        <select class="form-select form-select-sm mb-3" aria-label="Large select example" v-model="execTime">
          <option selected value="10">10 seconds</option>
          <option value="25">25 seconds</option>
          <option value="300">5 minutes</option>
        </select>
      </div>
    </div>
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

    <div v-for="(item, index) in pods" :key="index" style="margin-bottom: 2rem;">
      <div style="font-weight: bold">{{item.hostName}}</div>
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
    <LoginModal :id="loginId" ref="loginModal"/>
  </div>
</template>

<script setup>
import {ref, onMounted,onUnmounted,watch} from "vue";
import { useRouter } from 'vueRouter';
import Dashboard from "./components/dashboard.vue";
import LoginModal from "./components/loginModal.vue";

const [line1,line2,useR,homeEle] = [ref(null),ref(null),useRouter(),ref(null)];
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const [execTime,sseUrl] = [ref(10),ref("")];

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

watch(()=>execTime.value,(n,o)=>{
  execTime.value = n;
  sseConnect();
})

function sseConnect(){
  if(sse.value){
    sse.value.close();
  }
  sseUrl.value = `dashboard?time=${execTime.value}`;
  sse.value = sseApi.Init(sseUrl.value);
  sse.value.onopen = () => {
    console.log("connect success")
  }
  sse.value.addEventListener("dashboard",function (res) {
    const {code,msg,data} = JSON.parse(res.data);
    if (code === "1004"){
        loginModal.value.error(new Error(msg));
        sse.value.close();
        return
    }

    queue_total.value = data.queue_total;
    db_size.value = data.db_size;
    num_cpu.value = data.num_cpu;
    fail_count.value = data.fail_count;
    success_count.value = data.success_count;

    let npods = [];
    for(let key in data?.pods){
      if(key % 2 === 0){
        npods.push(JSON.parse(data.pods[key]));
      }
    }
    pods.value = npods;

    queuedMessagesOption.value = dashboardApi.QueueLine(data.queues,execTime.value);
    messageRatesOption.value = dashboardApi.MessageRateLine(data.queues,execTime.value);
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