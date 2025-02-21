<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of DeadLetter Log</h5>
      </div>
    </div>
    <div class="dlq">
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
      <table class="table table-striped table-hover">
        <thead>
        <tr>
          <th scope="col">#</th>
          <th scope="col">Id</th>
          <th scope="col">Channel</th>
          <th scope="col">Topic</th>
          <th scope="col">Mood Type</th>
          <th scope="col">AddTime</th>
          <th scope="col">Payload</th>
          <th scope="col">Action</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="(item, key) in logs" :key="key" style="height: 3rem;line-height:3rem">
          <th scope="row">{{item._id}}</th>
          <td><router-link to="" class="nav-link text-primary" style="display: contents" v-on:click="detailDlq(item)">{{item.id}}</router-link></td>
          <td>{{item.channel}}</td>
          <td>{{item.topic}}</td>
          <td>{{item.moodType}}</td>
          <td>{{item.addTime}}</td>
          <td>
              <span class="d-block text-truncate" style="max-width: 30rem;">
                <pre><code>    {{item.payload}}</code></pre>
              </span>
          </td>
          <td class="text-center text-nowrap">
            <RetryIcon @action="retryModal(item)" style="margin: 0 .25rem"/>
            <DeleteIcon @action="deleteModal(item)" style="margin:0 .25rem;"/>
          </td>
        </tr>
        </tbody>

      </table>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
    </div>
    <Action :label="retryLabel" :id="showRetryModal" :data-id="dataId" :warning="retryWarningHtml" :info="retryInfoHtml" @action="retryInfo">
      <template #title="{title}">
      </template>
    </Action>
    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="dataId" @action="deleteInfo">
      <template #title="{title}">
      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
  </div>
</template>
<script setup>
import { ref,onMounted } from "vue";
import { useRouter,useRoute } from 'vueRouter';
import Pagination from "../../components/pagination.vue";
import RetryIcon from "../../components/icons/retry_icon.vue";
import DeleteIcon from "../../components/icons/delete_icon.vue";
import Action from "../../components/action.vue";
import Btoast from "../../components/btoast.vue";

const [id,toastRef] = [ref("userToast"),ref(null)];
const [page,pageSize,total,cursor,logs] = [ref(1),ref(10),ref(1),ref(0),ref([])];
const [retryWarningHtml,retryInfoHtml] = [
  ref("Warning: Item retry cannot be undone!<br/> Please proceed with caution!"),
  ref("This operation will permanently retry the data of log.<br>\n" +
      "To prevent accidental actions, please confirm by entering the following:<br/>")
]
const [retryLabel,showRetryModal,dataId,retryItem] = [ref("retryLabel"),ref("showRetryModal"),ref(""),ref({})];
const [deleteLabel,showDeleteModal,deleteId] = [ref("deleteLabel"),ref("showDeleteModal"),ref("")]

async function dlqLogs() {
  let res = await dlqApi.List(page.value,pageSize.value);
  const {code,msg,data} = res;
  if(code !== "0000"){
    toastRef.value.show(msg);
    return;
  }
  logs.value = data.data;
  total.value = data.total;
  page.value =  data.cursor;
  cursor.value = data.cursor;
}

onMounted( ()=>{
  dlqLogs();
})
const [uRouter,route] = [useRouter(),useRoute()];
function detailDlq(item){
  uRouter.push("/admin/log/dlq/detail/"+item._id);
}

function retryModal(item){
  retryItem.value = {};
  dataId.value = "";
  const eleRetry = document.getElementById("showRetryModal");
  retryModal.value = new bootstrap.Modal(eleRetry);
  retryModal.value.show(eleRetry);
  retryItem.value = item;
  dataId.value = item._id;
}

async function retryInfo(){
  retryModal.value.hide();
  if(dataId.value === ""){
    toastRef.value.show("Missing Id");
    return;
  }
  try{
    let res = await dlqApi.Retry(dataId.value,retryItem.value);
    toastRef.value.show(res.msg);
    if(res.code === "0000"){
      await dlqLogs();
      return;
    }
  }catch (e) {
    toastRef.value.show(e.error);
  }
}

function deleteModal(item){
  deleteId.value = "";
  dataId.value = "";
  const ele = document.getElementById("showDeleteModal");
  deleteModal.value = new bootstrap.Modal(ele);
  deleteModal.value.show(ele);
  deleteId.value = item._id;
  dataId.value = item._id;
}

async function deleteInfo(){
  deleteModal.value.hide();
  if(deleteId.value === ""){
    toastRef.value.show("Missing Id");
    return;
  }
  try {
    let res = await dlqApi.Delete(deleteId.value);
    toastRef.value.show(res.msg);
    if(res.code === "0000"){
      await dlqLogs();
    }
  }catch (e) {
    toastRef.value.show(e.error);
  }
}

function changePage(pageVal,cursorVal){
  page.value = pageVal;
  cursor.value = cursorVal;
  sessionStorage.setItem("page",pageVal);
  dlqLogs();
}

</script>
<style scoped>
.dlq{
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>