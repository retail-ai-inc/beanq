<template>
  <div class="card">
    <div class="card-header bg-secondary text-white">
      <h5>Key Space</h5>
    </div>
    <table class="table table-striped table-hover">
      <thead>
      <tr>
        <th scope="col" class="text-start pl-10">DbName</th>
        <th scope="col" class="text-end">Keys</th>
        <th scope="col" class="text-end">Expires</th>
        <th scope="col" class="text-end pr-10">Avg ttl</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item,key) in keyspace" :key="key">
        <td class="text-start pl-10 text-bold">{{item.dbname}}</td>
        <td class="text-end">{{item.keys}}</td>
        <td class="text-end">{{item.expires}}</td>
        <td class="text-end pr-10">{{item.avg_ttl}}</td>
      </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup>
import {defineProps} from "vue";

const props = defineProps({
  keyspace:Array
})

</script>
<style>
.pl-10 {
  padding-left: 10px
}
.pr-10 {
  padding-right: 10px
}
.text-bold {
  font-weight: bold;
}
</style>