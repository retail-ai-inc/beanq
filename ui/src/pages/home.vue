<template>

  <div class="home" ref="homeEle">
    <div class="row justify-content-end">
      <div class="col-1">
        <select class="form-select form-select-sm mb-3" aria-label="Large select example" v-model="execTime">
          <option selected value="300">5 minute</option>
          <option value="1800">30 minute</option>
          <option value="7200">2 hour</option>
          <option value="18000">5 hour</option>
          <option value="43200">12 hour</option>
          <option value="86400">1 day</option>
          <option value="172800">2 day</option>
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
      <Dashboard />
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
const [execTime,sseUrl] = [ref(300),ref("")];

let [
    queuedMessagesOption,
    messageRatesOption,
    nodeId,
    sse,
    resizeObserver,
    pods
  ] = [ref({}),ref({}),ref(""),ref(null),null,ref({})];


function resize(){
  [line1.value,line2.value].forEach(chart=>{
    chart?.resize();
  })
}

const queues = ref([]);
watch(()=>execTime.value,(n,o)=>{
  execTime.value = n;
  queues.value = [];
  sseConnect();
})

function sseConnect(){
  if(sse.value){
    sse.value.close();
  }
  sseUrl.value = `dashboard/graphic?time=${execTime.value}`;
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
    if(code === "1111"){
      sse.value.close();
      return;
    }

    let newdata = data.map(item=>{
      try {
        return JSON.parse(item);
      }catch (e) {
        return null;
      }
    })
    newdata = newdata.filter(item=>item !== null);
    queues.value.push(...newdata);
    queuedMessagesOption.value = dashboardApi.QueueLine(queues.value,execTime.value);
    messageRatesOption.value = dashboardApi.MessageRateLine(queues.value,execTime.value);
  })
  sse.value.onerror = (err)=>{
    console.log(err)
    sse.value.close();
    setTimeout(sseConnect,1500);
  }
}

const getPods = async() => {
  try {
    let res = await dashboardApi.Pods();
    let npods = res.map(item=>{
      try {
        return JSON.parse(item);
      }catch (e) {
        return null;
      }
    });
    pods.value = npods.filter(item=>item !== null);
  }catch (e) {

  }
}

onMounted( () => {

  let observerFun = ()=>{
    resize();
    return false;
  }
  resizeObserver = new ResizeObserver((entries)=>{
    Base.Debounce( observerFun(),5000);
  });
  const parentEle = homeEle.value.parentElement;
  resizeObserver.observe(parentEle);
  getPods();
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