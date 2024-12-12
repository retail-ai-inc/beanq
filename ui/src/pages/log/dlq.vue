<template>
  <div class="dlq">
    <table class="table table-striped table-hover">
      <thead>
      <tr>
        <th scope="col">#</th>
        <th scope="col">Id</th>
        <th scope="col">Channel</th>
        <th scope="col">Topic</th>
        <th scope="col">MoodType</th>
        <th scope="col">Status</th>
        <th scope="col">AddTime</th>
        <th scope="col">Payload</th>
        <th scope="col">Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item, key) in dlqLogs" :key="key" style="height: 3rem;line-height:3rem">
        <th scope="row">{{parseInt(key)+1}}</th>
        <td><router-link to="" class="nav-link text-primary" style="display: contents" v-on:click="detailDlq(item)">{{item.id}}</router-link></td>
        <td>{{item.channel}}</td>
        <td>{{item.topic}}</td>
        <td>{{item.moodType}}</td>
        <td>
          <span v-if="item.status == 'success'" class="text-success">{{item.status}}</span>
          <span v-else-if="item.status == 'failed'" class="text-danger">{{item.status}}</span>
          <span v-else-if="item.status == 'published'" class="text-warning">{{item.status}}</span>
        </td>
        <td>{{item.addTime}}</td>
        <td>
              <span class="d-block text-truncate" style="max-width: 30rem;">
                {{item.payload}}
              </span>
        </td>
        <td>
          <div class="btn-group-sm" role="group">
            <button type="button" class="btn btn-primary dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">
              actions
            </button>
            <ul class="dropdown-menu">
              <!--v-if="item.status == 'failed'"-->
              <li ><a class="dropdown-item" href="javascript:;" @click="retryItem(item)">Retry</a></li>
              <li><a class="dropdown-item" href="javascript:;" @click="deleteItem(item)">Delete</a></li>
<!--              <li><a class="dropdown-item" href="javascript:;" @click="editModal(item)">Edit Payload</a></li>-->
            </ul>
          </div>
        </td>
      </tr>
      </tbody>

    </table>
  </div>
</template>
<script setup>
import { reactive,onMounted,toRefs,onUnmounted } from "vue";

let data = reactive({
  dlqLogs:[]
})

onMounted(async ()=>{
  let res = await dlqApi.List();
  data.dlqLogs = res.data;
})

function detailDlq(item){

}

function retryItem(item){

}

function deleteItem(item){

}

const {dlqLogs} = toRefs(data);
</script>
<style scoped>
.dlq{
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>