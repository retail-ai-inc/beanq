<template>
  <div class="opt-log">
    <table class="table table-striped table-hover">
      <thead>
      <tr>
        <th scope="col">#</th>
        <th scope="col">Account</th>
        <th scope="col">Visit</th>
        <th scope="col">Data</th>
        <th scope="col">Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item, key) in list" :key="key" style="height: 3rem;line-height:3rem">
        <th scope="row" style="width: 5%">{{parseInt(key)+1}}</th>
        <td style="width: 15%">{{item.user}}</td>
        <td style="width: 20%"><span class="d-inline-block text-truncate" style="">{{item.uri}}</span></td>
        <td style="width: 55%">
          <span class="d-inline-block text-truncate" style="">
            {{item.data}}
          </span>
        </td>
        <td style="width: 5%">
          <a href="javascript:;" @click="deleteLog(item)">Delete</a>
        </td>
      </tr>
      </tbody>
    </table>
  </div>
</template>
<script setup>
import { reactive,onMounted,toRefs,onUnmounted } from "vue";

let data = reactive({
  list:[],
})

onMounted(async ()=>{

  let res = await logApi.OptLog(0,10);
  data.list = res.data;

})

onUnmounted(()=>{
})

async function deleteLog(item){

  let res = await logApi.Delete(item.account);
  if(res.code == "0000"){
    let res = await logApi.OptLog();
    data.list = res.data;
    return
  }
}

const {list} = toRefs(data);
</script>

<style scoped>
.user{
  transition: opacity 0.5s ease;
  opacity: 1;
}
.green{
  color:var(--bs-success);
}
.red{
  color:var(--bs-danger);
}
</style>