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
  </div>
</template>

<script setup>
import {ref, onMounted,onUnmounted,inject} from "vue";
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
    resizeObserver
  ] = [ref(0),ref(0),ref(0),ref(0),ref(0),ref({}),ref({}),ref(""),ref(null),null];


function resize(){
  [line1.value,line2.value].forEach(chart=>{
    chart?.resize();
  })
}

onMounted( () => {

  const parentEle = homeEle.value.parentElement;
  resizeObserver = new ResizeObserver((entries)=>{
    resize();
  })
  Base.Debounce( resizeObserver.observe(parentEle),300) ;

  if(sse.value){
    sse.value.close();
  }
  sse.value = sseApi.Init("dashboard");
  sse.value.onopen = () => {
    console.log("success")
  }
  sse.value.addEventListener("dashboard",function (res) {
    let result = JSON.parse(res.data);
    if (result.code !== "0000"){
      return
    }

    queue_total.value = result.data.queue_total;
    db_size.value = result.data.db_size;
    num_cpu.value = result.data.num_cpu;
    fail_count.value = result.data.fail_count;
    success_count.value = result.data.success_count;

    queuedMessagesOption.value = dashboardApi.QueueLine(result.data.queues);
    messageRatesOption.value = dashboardApi.MessageRateLine(result.data.queues);
  })
  sse.value.onerror = (err)=>{
    useR.push("/login");
    console.log(err)
  }

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