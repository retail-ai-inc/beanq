<template>
  <div class="card">
    <div class="card-header bg-secondary text-white">
      <h5>Command</h5>
    </div>
    <div class="card-body" style="padding: 0">
      <div class="table-responsive">
        <table class="table table-striped table-hover">
          <thead>
          <tr>
            <th scope="col" class="text-start pl-10">Command</th>
            <th scope="col" class="text-end">Calls</th>
            <th scope="col" class="text-end">Usec</th>
            <th scope="col" class="text-end pr-10">UsecPerCall</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="(v,k) in commands" :key="k">
            <td class="text-start pl-10 text-bold">{{v.command}}</td>
            <td class="text-end">{{v.calls}}</td>
            <td class="text-end">{{v.usec}}</td>
            <td class="text-end pr-10">{{v.usec_per_call}}</td>
          </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps} from "vue";

const props = defineProps({
  commands:Array
})

</script>
<style>
.table {
  margin-bottom: 0 !important;
}
.pl-10 {
  padding-left: 0.625rem
}
.pr-10 {
  padding-right: 0.625rem
}
.text-bold {
  font-weight: bold;
}

</style>