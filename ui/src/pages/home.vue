<template>

  <div class="home" ref="homeEle">
<!--    <div class="row">-->
<!--      <div class="col d-flex">-->
<!--        <div class="btn-group" role="group" aria-label="Basic radio toggle button group">-->
<!--          <input type="radio" class="btn-check" name="btnradio" id="btnradio1" autocomplete="off" checked>-->
<!--          <label class="btn btn-outline-primary" for="btnradio1">5 minute</label>-->

<!--          <input type="radio" class="btn-check" name="btnradio" id="btnradio2" autocomplete="off">-->
<!--          <label class="btn btn-outline-primary" for="btnradio2">10 minute</label>-->

<!--          <input type="radio" class="btn-check" name="btnradio" id="btnradio3" autocomplete="off">-->
<!--          <label class="btn btn-outline-primary" for="btnradio3">30 minute</label>-->

<!--          <input type="radio" class="btn-check" name="btnradio" id="btnradio4" autocomplete="off">-->
<!--          <label class="btn btn-outline-primary" for="btnradio4">6 hour</label>-->

<!--          <input type="radio" class="btn-check" name="btnradio" id="btnradio5" autocomplete="off">-->
<!--          <label class="btn btn-outline-primary" for="btnradio5">12 hour</label>-->

<!--          <input type="radio" class="btn-check" name="btnradio" id="btnradio6" autocomplete="off">-->
<!--          <label class="btn btn-outline-primary" for="btnradio6">24 hour</label>-->
<!--        </div>-->
<!--        <div style="width: 25rem;">-->
<!--          <vue-date-picker range v-model="date" multi-calendars></vue-date-picker>-->
<!--        </div>-->
<!--      </div>-->
<!--    </div>-->
    <div class="row justify-content-end">
      <div class="col-1">
        <select class="form-select form-select-sm mb-3" aria-label="Large select example" v-model="execTime">
          <option selected value="5m">5 minute</option>
          <option value="10m">10 minute</option>
          <option value="30m">30 minute</option>
          <option value="6h">6 hour</option>
          <option value="12h">12 hour</option>
          <option value="24h">24 hour</option>
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
const [execTime,sseUrl] = [ref("5m"),ref("")];

let [
    queuedMessagesOption,
    messageRatesOption,
    nodeId,
    sse,
    resizeObserver,
    pods
  ] = [ref({}),ref({}),ref(""),ref(null),null,ref([])];

const date = ref([new Date(),new Date()]);

function resize(){
  [line1.value,line2.value].forEach(chart=>{
    chart?.resize();
  })
}

const [queues,queuesCount] = [ref([]),ref(0)];
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
        return;
    }
    if(code === "1111" && msg === "DONE"){
      queuesCount.value = data;
      sse.value.close();
    }
    if (typeof data !== "object"){

      queuedMessagesOption.value = dashboardApi.QueueLine(queues.value,execTime.value,queuesCount.value);
      messageRatesOption.value = dashboardApi.MessageRateLine(queues.value,execTime.value,queuesCount.value);
    }else{
      queues.value.push(...data);
    }


  })
  sse.value.onerror = (err)=>{
    console.log(err)
    sse.value.close();
    setTimeout(sseConnect,1500);
  }
}
let [ssePod] = [ref(null)];
function getPods(){

  if(ssePod.value){
    ssePod.value.close();
  }
  ssePod.value = sseApi.Init(`dashboard/pods`);
  ssePod.value.onopen = () => {
    console.log("connect success")
  }
  ssePod.value.addEventListener("pods",function (res) {

    const {code,msg,data} = JSON.parse(res.data);

    if (code === "1004"){
      loginModal.value.error(new Error(msg));
      sse.value.close();
      return;
    }

    pods.value = data.map(item=>{
      try{
        return JSON.parse(item);
      }catch (e){
        return undefined
      }
    }).filter(item=>(item !== undefined) && (item !== null) && (item !== "") && (Object.keys( item).length !== 0))
  })
  ssePod.value.onerror = (err)=>{
    console.log(err)
    ssePod.value.close();
    setTimeout(ssePod,1500);
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
  if(ssePod.value){
    ssePod.value.close();
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