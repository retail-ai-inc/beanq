<template>
  <div class="workflow">
    <div class="container-fluid">
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
      <div class="row">
        <div class="col-12">
          <div class="table-responsive">
            <table class="table table-striped table-hover">
              <thead>
              <tr>
                <th scope="col">#</th>
                <th scope="col">Id</th>
                <th scope="col">Channel</th>
                <th scope="col">Topic</th>
                <th scope="col">Mood Type</th>
                <th scope="col">Status</th>
                <th scope="col">AddTime</th>
                <th scope="col">Payload</th>
              </tr>
              </thead>
              <tbody>
              <tr v-for="(item, key) in workflowlogs" :key="key" style="height: 3rem;line-height:3rem">
                <th scope="row">{{parseInt(key)+1}}</th>
                <td><router-link to="" class="nav-link text-primary" style="display: contents" v-on:click="detailEvent(item)">{{item.id}}</router-link></td>
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
              </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>

    </div>
  </div>
</template>
<script setup>
import { ref,onMounted,onUnmounted,computed } from "vue";
import Pagination from "../../components/pagination.vue";

const [workflowlogs,page,pageSize,total,cursor] = [ref([]),ref(1),ref(10),ref(0),ref(0)]
// paging
async function changePage(pageo,cursoro){
  page.value = pageo;
  cursor.value = cursoro;
  sessionStorage.setItem("page",page)
}


onMounted(async()=>{
  page.value = sessionStorage.getItem("page")??1;
  let res = await logApi.WorkFlowLogs(page.value,pageSize.value)
  console.log(res);
})

onUnmounted(()=>{

})

</script>
<style scoped>
.workflow{
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>