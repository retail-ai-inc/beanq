<template>

    <div class="d-flex justify-content-between">
        <div style="width:50%">
          <v-chart class="chart" :option="barOption" />
        </div>
        <div style="width:50%">
          <v-chart class="chart" :option="lineOption"/>
        </div>
    </div>

</template>

<script setup>
import {ref} from "vue";
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
      data: [120, 200, 150, 80, 70, 110, 130],
      type: 'bar',
      showBackground: true,
      backgroundStyle: {
        color: 'rgba(180, 180, 180, 0.2)'
      }
    }
  ]
});
const lineOption = ref({
  title: {
    text: 'Tasks Processed'
  },
  tooltip: {
    trigger: 'axis'
  },
  legend: {
    bottom:"10",
    data: [
        {
          name:"succeed",
          lineStyle: {
            color: '#198754'
          },
          itemStyle:{
            color:"#198754"
          }
        },
        {
          name:"failed",
          lineStyle:{
            color:'#dc3545'
          },
          itemStyle: {
            color:'#dc3545'
          }
        }
    ]
  },
  toolbox: {
    feature: {
      //saveAsImage: {}
    }
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
  },
  yAxis: {
    type: 'value'
  },
  series: [
    {
      name: 'succeed',
      type: 'line',
      lineStyle: {
        color: '#198754'
      },
      itemStyle:{
        color:"#198754"
      },
      data:[220, 182, 191, 234, 290, 330, 310]
    },
    {
      name: 'failed',
      type: 'line',
      lineStyle:{
        color:'#dc3545'
      },
      itemStyle:{
        color:"#dc3545"
      },
      data:[120, 132, 101, 134, 90, 230, 210]
    }
  ]
})
</script>
<style scoped>
.chart{
  width:100%;height:80vh;
}
</style>