<template>
  <div class="row align-items-start dashboard-index" style="margin: 1.25rem 0;color:#fff;">
    <div class="col bg-primary">
      <div class="inner">
        <h3>
          <router-link to="/admin/queue" class="nav-link text-muted link-color" >{{queue_total}}</router-link>
        </h3>
        <h5 class="my-auto">Queue Total</h5>
      </div>
      <div class="small-box">
        <QueueTotalIcon />
      </div>
    </div>
    <div class="col bg-success">
      <div class="inner">
        <h3>
          <router-link to="/admin/redis" class="nav-link text-muted link-color">{{num_cpu}}</router-link>
        </h3>
        <h5 class="my-auto">CPU Total</h5>
      </div>
      <div class="small-box">
        <CpuTotalIcon />
      </div>
    </div>
    <div class="col bg-danger">
      <div class="inner">
        <h3>
          <router-link to="log/event?status=failed" class="nav-link text-muted link-color">{{fail_count}}</router-link>
        </h3>
        <h5 class="my-auto">Fail Total</h5>
      </div>
      <div class="small-box">
        <FailTotalIcon />
      </div>
    </div>
    <div class="col bg-info">
      <div class="inner">
        <h3>
          <router-link to="log/event?status=success" class="nav-link text-muted link-color">{{success_count}}</router-link>
        </h3>
        <h5 class="my-auto">Success Total</h5>
      </div>
      <div class="small-box">
        <SuccessTotalIcon />
      </div>
    </div>
    <div class="col bg-warning">
      <div class="inner">
        <h3>
          <router-link to="db-size" class="nav-link text-muted link-color">{{db_size}}</router-link>
        </h3>
        <h5 class="my-auto">DB Size</h5>
      </div>
      <div class="small-box">
        <TotalIcon />
      </div>
    </div>
  </div>
</template>
<script setup>
import {ref,onMounted} from "vue";
import QueueTotalIcon from "./icons/queue_total_icon.vue";
import CpuTotalIcon from "./icons/cpu_total_icon.vue";
import FailTotalIcon from "./icons/fail_total_icon.vue";
import SuccessTotalIcon from "./icons/success_total_icon.vue";
import TotalIcon from "./icons/total_icon.vue";

const [queue_total,num_cpu,fail_count,success_count,db_size] = [ref(0),ref(0),ref(0),ref(0),ref(0)];

onMounted(async ()=>{
  try {
    let res = await dashboardApi.Total();
    queue_total.value = res?.queue_total || 0;
    num_cpu.value = res?.num_cpu || 0;
    fail_count.value = res?.fail_count || 0;
    success_count.value = res?.success_count || 0;
    db_size.value = res?.db_size || 0;
  }catch (e) {

  }
})

</script>
<style scoped>
.dashboard-index .col{
  height:7.5rem;
  padding:1rem;
  position: relative;
}
.link-color{
  display: inline-block;
  color: #fff !important;
}
.small-box {
  opacity: 0.15;
  z-index: 0;
}
</style>