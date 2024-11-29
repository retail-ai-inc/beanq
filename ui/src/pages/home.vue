<template>

  <div class="home">
    <div class="chart-container">
      <div class="chart-h">
        <v-chart class="chart" :option="queuedMessagesOption"/>
      </div>
      <div class="chart-h">
        <v-chart class="chart" :option="refererOption"/>
      </div>
      <div class="chart-h">
        <v-chart class="chart" :option="messageRatesOption"/>
      </div>
      <div class="chart-h">
        <v-chart class="chart" :option="barOption"/>
      </div>
    </div>

    <div class="container-fluid" style="margin-bottom: 40px">
      <Dashboard :queue_total="queue_total"
                 :num_cpu="num_cpu"
                 :fail_count="fail_count"
                 :success_count="success_count"
                 :db_size="db_size"/>
    </div>

    <div class="container-fluid text-center">
      <div class="row justify-content-between">
        <div class="col-4">
          <Command :commands="commands"/>
          <KeySpace :keyspace="keyspace" class="mt-1"/>
        </div>
        <div class="col-4">
          <Client :clients="clients"/>
          <Memory :memory="memory" class="mt-1"/>
        </div>
        <div class="col-4">
          <Stats :stats="stats"/>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import {ref, reactive, onMounted, toRefs,} from "vue";
import Dashboard from "./components/dashboard.vue";
import Command from "./components/command.vue";
import Client from "./components/client.vue";
import Memory from "./components/memory.vue";
import KeySpace from "./components/keySpace.vue";
import Stats from "./components/stats.vue";


let data = reactive({
  "queue_total": 0,
  "db_size": 0,
  "num_cpu": 0,
  "fail_count": 0,
  "success_count": 0,
  "commands": [],
  "clients": {},
  "stats": {},
  "keyspace": [],
  "memory": {}
})

function getTotal() {
  return request.get("dashboard");
}

onMounted(async () => {

  let total = await getTotal();

  Object.assign(data, total.data);
  data.commands = total.data.commands;
  data.clients = total.data.clients;
  data.stats = total.data.stats;
  data.keyspace = total.data.keyspace;
  data.memory = total.data.memory;
})

const queuedMessagesOption = ref({
  title: {
    text: 'Queued messages',
    subtext: '(chart:last minute)(?)',
  },
  tooltip: {
    trigger: 'axis'
  },
  legend: {
    data: ['Ready', 'Unacked', 'Total']
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true
  },
  toolbox: {
    feature: {
      // saveAsImage: {}
    }
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: ['09:02:10', '09:02:20', '09:02:30', '09:02:40', '09:02:50', '09:03:00']
  },
  yAxis: {
    type: 'value',
    axisLine: {
      show: true,
    }

  },
  series: [
    {
      name: 'Ready',
      type: 'line',
      data: [120, 132, 101, 134, 90, 230]
    },
    {
      name: 'Unacked',
      type: 'line',
      data: [220, 182, 191, 234, 290, 330]
    },
    {
      name: 'Total',
      type: 'line',
      data: [150, 232, 201, 154, 190, 330]
    }
  ]
});

const messageRatesOption = ref({
  title: {
    text: 'Message rates',
    subtext: '(chart:last minute)(?)',
  },

  tooltip: {
    trigger: 'axis'
  },
  legend: {
    data: ['Publish', 'Confirm', 'Deliver', 'Redelivered', 'Acknowledge', 'Get', 'Get(noack)']
  },
  grid: {
    left: '3%',
    right: '4%',
    bottom: '3%',
    containLabel: true
  },
  toolbox: {
    feature: {
      // saveAsImage: {}
    }
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: ['09:02:10', '09:02:20', '09:02:30', '09:02:40', '09:02:50', '09:03:00']
  },
  yAxis: {
    type: 'value',
    axisLine: {
      show: true,
    },
    axisLabel: {
      formatter: function (value) {
        return value + '/s';
      }
    }
  },
  series: [
    {
      name: 'Publish',
      type: 'line',
      data: [120, 132, 101, 134, 90, 230]
    },
    {
      name: 'Confirm',
      type: 'line',
      data: [220, 182, 191, 234, 290, 330]
    },
    {
      name: 'Deliver',
      type: 'line',
      data: [150, 232, 201, 154, 190, 330]
    },
    {
      name: 'Redelivered',
      type: 'line',
      data: [320, 332, 301, 334, 390, 330]
    },
    {
      name: 'Acknowledge',
      type: 'line',
      data: [820, 932, 901, 934, 1290, 1330]
    },
    {
      name: 'Get',
      type: 'line',
      data: [820, 932, 901, 934, 1290, 1330]
    },
    {
      name: 'Get(noack)',
      type: 'line',
      data: [820, 932, 901, 934, 1290, 1330]
    }
  ]
});

const barOption = ref({
  title: {
    text: 'Queue Size',
    left: 'left'
  },
  xAxis: {
    type: 'category',
    data: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
  },
  yAxis: {
    type: 'value'
  },
  series: [
    {
      data: [120, 200, 350, 420, 170, 210, 130],
      type: 'bar',
      showBackground: true,
      backgroundStyle: {
        color: 'rgba(180, 180, 180, 0.2)'
      }
    }
  ]
});
const refererOption = ref({
  title: {
    text: 'Referer of a Website',
    subtext: 'Fake Data',
    left: 'center'
  },
  tooltip: {
    trigger: 'item'
  },
  legend: {
    orient: 'vertical',
    left: 'left'
  },
  series: [
    {
      name: 'Access From',
      type: 'pie',
      radius: '50%',
      data: [
        { value: 1048, name: 'Search Engine' },
        { value: 735, name: 'Direct' },
        { value: 580, name: 'Email' },
        { value: 484, name: 'Union Ads' },
        { value: 300, name: 'Video Ads' }
      ],
      emphasis: {
        itemStyle: {
          shadowBlur: 10,
          shadowOffsetX: 0,
          shadowColor: 'rgba(0, 0, 0, 0.5)'
        }
      }
    }
  ]
});

const {
  queue_total,
  db_size,
  num_cpu,
  fail_count,
  success_count,
  commands,
  clients,
  stats,
  keyspace,
  memory
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