<template>
  <div class="opt-log">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of Operation Logs</h5>
      </div>
    </div>

    <Spinner v-if="loading"/>
    <div v-else>
      <NoMessage v-if="list.length <= 0"/>
      <div v-else>
        <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
        <table class="table table-striped table-hover"  style="table-layout: auto;">
          <thead>
          <tr>
            <th scope="col" class="w-table-number">#</th>
            <th scope="col" class="text-nowrap">Id</th>
            <th scope="col" class="text-nowrap">Add Time</th>
            <th scope="col" class="text-nowrap">Account</th>
            <th scope="col" class="text-nowrap">Visit</th>
            <th scope="col" class="text-nowrap">Data</th>
            <th scope="col" class="text-center">Action</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="(item, key) in list" :key="key" style="height: 2.5rem;">
            <td class="text-right">{{parseInt(key)+1}}</td>
            <td>{{item._id}}</td>
            <td>
              <pre><code>{{item.addTime}}</code></pre>
            </td>
            <td>{{item.user}}</td>
            <td>
          <span class="d-inline-block text-truncate" style="max-width: 50rem">
            <pre><code>{{item.uri}}</code></pre>
          </span>
            </td>
            <td>
          <span class="d-inline-block text-truncate" style="max-width: 400px">
            {{item.data}}
          </span>
            </td>
            <td class="text-center text-nowrap">
              <Delete_icon @action="deleteShowModal(item)"/>
            </td>
          </tr>
          </tbody>
        </table>
        <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
      </div>
    </div>

    <Btoast :id="id" ref="toastRef"/>
    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="mid" @action="deleteLog">
      <template #title="{title}">
        Are you sure to delete?
      </template>
    </Action>
    <LoginModal :id="loginId" ref="loginModal"/>
  </div>
</template>
<script setup>
import { ref,onMounted,onUnmounted } from "vue";
import Delete_icon from "../components/icons/delete_icon.vue";
import Btoast from "../components/btoast.vue";
import Action from "../components/action.vue";
import Pagination from "../components/pagination.vue";
import LoginModal from "../components/loginModal.vue";
import Spinner from "../components/spinner.vue";
import NoMessage from "../components/noMessage.vue";

const [list,id,toastRef] = [ref([]),ref("optLog"),ref(null)];
const [deleteLabel,showDeleteModal,deleteModal,mid] = [ref("deleteLabel"),ref("showDeleteModal"),ref(null),ref("")];
const [page,pageSize,total,cursor] = [ref(1),ref(10),ref(0),ref(1)];

const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const loading = ref(false);

const getOptLogs = (async (pageV,pageSizev)=>{
  loading.value = true;
  try {
    let res = await logApi.OptLog(pageV,pageSizev);
    list.value = res.data;
    total.value = Math.ceil(res.total / pageSize.value);
    cursor.value = res.cursor;
    setTimeout(()=>{
      loading.value = false;
    },800);
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e);
  }

})
onMounted( ()=>{
  getOptLogs(page.value,pageSize.value);
})

onUnmounted(()=>{
})

function deleteShowModal(item){
  mid.value = "";
  const ele = document.getElementById("showDeleteModal");
  deleteModal.value = new bootstrap.Modal(ele);
  deleteModal.value.show(ele);
  mid.value = item._id;
}

async function deleteLog(){

  try {
    let res = await logApi.DeleteOptLog(mid.value);
    deleteModal.value.hide();
    toastRef.value.show("success");
    await getOptLogs(page.value,pageSize.value);
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e.message);
  }
}

async function changePage(pageo,cursoro){

  page.value = pageo;
  let res = await logApi.OptLog(page.value,pageSize.value);
  list.value = res.data;
  total.value = Math.ceil(res.total / pageSize.value);
  cursor.value = cursoro;
}

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