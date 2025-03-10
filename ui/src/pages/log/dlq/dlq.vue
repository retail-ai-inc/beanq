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
          <th scope="row">
            <Copy :text="item._id" />
          </th>
          <td><router-link to="" class="nav-link text-primary" style="display: contents" v-on:click="detailDlq(item)">{{maskString(item.id)}}</router-link></td>
          <td>{{item.channel}}</td>
          <td>{{item.topic}}</td>
          <td>{{item.moodType}}</td>
          <td>
            <TimeToolTips :past-time="item.addTime"/>
          </td>
          <td>
            <More :payload="item.payload"/>
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
    <Btoast :id="id" ref="toastRef" />

    <LoginModal :id="noticeId" ref="loginModal"/>
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
import LoginModal from "../../components/loginModal.vue";
import TimeToolTips from "../../components/timeToolTips.vue";
import More from "../../components/more.vue";
import Copy from "../../components/copy.vue";
import i18n from "i18n";


const [id,toastRef] = [ref("userToast"),ref(null)];
const [page,pageSize,total,cursor,logs] = [ref(1),ref(10),ref(0),ref(0),ref([])];
const [retryWarningHtml,retryInfoHtml] = [
    ref(i18n.global.t('retryWarningHtml')),
    ref(i18n.global.t('retryInfoHtml'))
]
const [retryLabel,showRetryModal,dataId,retryItem] = [ref("retryLabel"),ref("showRetryModal"),ref(""),ref({})];
const [deleteLabel,showDeleteModal,deleteId] = [ref("deleteLabel"),ref("showDeleteModal"),ref("")];

const [noticeId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];

const maskString = ((id)=>{
  return Base.MaskString(id)
})

async function dlqLogs() {
  try {
    let res = await dlqApi.List(page.value,pageSize.value);
    const{cursor:resCursor,data,total:resTotal} = res;

    logs.value = data;
    total.value = resTotal;
    page.value =  resCursor;
    cursor.value = resCursor;
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
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
    toastRef.value.show("success");
    await dlqLogs();
  }catch (err) {
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    toastRef.value.show(err.error);
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
    toastRef.value.show("success");
    await dlqLogs();
  }catch (err) {
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    toastRef.value.show(err.error);
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