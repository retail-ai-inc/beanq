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
        <svg class="s-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 448 512">
          <path d="M0 96C0 78.3 14.3 64 32 64l384 0c17.7 0 32 14.3 32 32s-14.3 32-32 32L32 128C14.3 128 0 113.7 0 96zM0 256c0-17.7 14.3-32 32-32l384 0c17.7 0 32 14.3 32 32s-14.3 32-32 32L32 288c-17.7 0-32-14.3-32-32zM448 416c0 17.7-14.3 32-32 32L32 448c-17.7 0-32-14.3-32-32s14.3-32 32-32l384 0c17.7 0 32 14.3 32 32z"/>
        </svg>
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
        <svg class="s-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
          <path d="M176 24c0-13.3-10.7-24-24-24s-24 10.7-24 24l0 40c-35.3 0-64 28.7-64 64l-40 0c-13.3 0-24 10.7-24 24s10.7 24 24 24l40 0 0 56-40 0c-13.3 0-24 10.7-24 24s10.7 24 24 24l40 0 0 56-40 0c-13.3 0-24 10.7-24 24s10.7 24 24 24l40 0c0 35.3 28.7 64 64 64l0 40c0 13.3 10.7 24 24 24s24-10.7 24-24l0-40 56 0 0 40c0 13.3 10.7 24 24 24s24-10.7 24-24l0-40 56 0 0 40c0 13.3 10.7 24 24 24s24-10.7 24-24l0-40c35.3 0 64-28.7 64-64l40 0c13.3 0 24-10.7 24-24s-10.7-24-24-24l-40 0 0-56 40 0c13.3 0 24-10.7 24-24s-10.7-24-24-24l-40 0 0-56 40 0c13.3 0 24-10.7 24-24s-10.7-24-24-24l-40 0c0-35.3-28.7-64-64-64l0-40c0-13.3-10.7-24-24-24s-24 10.7-24 24l0 40-56 0 0-40c0-13.3-10.7-24-24-24s-24 10.7-24 24l0 40-56 0 0-40zM160 128l192 0c17.7 0 32 14.3 32 32l0 192c0 17.7-14.3 32-32 32l-192 0c-17.7 0-32-14.3-32-32l0-192c0-17.7 14.3-32 32-32zm192 32l-192 0 0 192 192 0 0-192z"/>
        </svg>
      </div>
    </div>
    <div class="col bg-danger">
      <div class="inner">
        <h3>
          <router-link to="" class="nav-link text-muted link-color">{{fail_count}}</router-link>
        </h3>
        <h5 class="my-auto">Fail Total</h5>
      </div>
      <div class="small-box">
        <svg class="s-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
          <path d="M256 512A256 256 0 1 0 256 0a256 256 0 1 0 0 512zm0-384c13.3 0 24 10.7 24 24l0 112c0 13.3-10.7 24-24 24s-24-10.7-24-24l0-112c0-13.3 10.7-24 24-24zM224 352a32 32 0 1 1 64 0 32 32 0 1 1 -64 0z"/>
        </svg>
      </div>
    </div>
    <div class="col bg-info">
      <div class="inner">
        <h3>
          <router-link to="" class="nav-link text-muted link-color">{{success_count}}</router-link>
        </h3>
        <h5 class="my-auto">Success Total</h5>
      </div>
      <div class="small-box">
        <svg class="s-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
          <path d="M256 512A256 256 0 1 0 256 0a256 256 0 1 0 0 512zM369 209L241 337c-9.4 9.4-24.6 9.4-33.9 0l-64-64c-9.4-9.4-9.4-24.6 0-33.9s24.6-9.4 33.9 0l47 47L335 175c9.4-9.4 24.6-9.4 33.9 0s9.4 24.6 0 33.9z"/>
        </svg>
      </div>
    </div>
    <div class="col bg-warning">
      <div class="inner">
        <h3>
          <router-link to="" class="nav-link text-muted link-color">{{db_size}}</router-link>
        </h3>
        <h5 class="my-auto">Total Payload</h5>
      </div>
      <div class="small-box">
        <svg class="s-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
          <path d="M498.1 5.6c10.1 7 15.4 19.1 13.5 31.2l-64 416c-1.5 9.7-7.4 18.2-16 23s-18.9 5.4-28 1.6L284 427.7l-68.5 74.1c-8.9 9.7-22.9 12.9-35.2 8.1S160 493.2 160 480l0-83.6c0-4 1.5-7.8 4.2-10.8L331.8 202.8c5.8-6.3 5.6-16-.4-22s-15.7-6.4-22-.7L106 360.8 17.7 316.6C7.1 311.3 .3 300.7 0 288.9s5.9-22.8 16.1-28.7l448-256c10.7-6.1 23.9-5.5 34 1.4z"/>
        </svg>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps} from "vue";

const props = defineProps({
  queue_total:0,
  num_cpu:0,
  fail_count:0,
  success_count:0,
  db_size:0
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
.s-icon {
  width: 55px;
  position: absolute;
  right: 20px;
  top: 20px;
}
</style>