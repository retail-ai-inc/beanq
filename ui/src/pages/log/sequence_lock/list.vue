<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of Sequence Lock</h5>
      </div>
    </div>
    <div class="dlq">
      <Search :form="form" @search="search"/>

      <Spinner v-if="loading" />
      <div v-else>
        <NoMessage v-if="logs.length <= 0"/>
        <div v-else>
          <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
          <table class="table table-striped table-hover">
            <thead>
            <tr>
              <th scope="col">Message Id</th>
              <th scope="col">Channel</th>
              <th scope="col">Topic</th>
              <th scope="col">Order Key</th>
              <th scope="col">Mood Type</th>
              <th scope="col">Add Time</th>
              <th scope="col">Payload</th>
              <th scope="col">Action</th>
            </tr>
            </thead>
            <tbody>
            <tr v-for="(item, key) in logs" :key="key" style="height: 3rem;line-height:3rem">
              <th scope="row">
                {{item.id}}
              </th>
              <td>{{item.channel}}</td>
              <td><div @click="filter(item.topic)" style="cursor: copy">{{item.topic}}</div></td>
              <td>{{item.orderKey}}</td>
              <td>{{item.moodType}}</td>
              <td>
                <TimeToolTips :past-time="item.addTime"/>
              </td>
              <td>
                <More :payload="item.payload"/>
              </td>
              <td class="text-center text-nowrap">
                <UnLockIcon @action="doUnlockModal(item)" style="margin:0 .25rem;"/>
<!--                <DetailIcon @action="detail(item)" style="margin:0 .25rem"/>-->
              </td>
            </tr>
            </tbody>

          </table>
          <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
        </div>
      </div>
    </div>
    <Action :label="unlockLabel" :id="showUnlockModal" :data-id="dataId" :warning="$t('retryWarningHtml')" :info="$t('unlockInfoHtml')" @action="unlockInfo">
      <template #title="{title}">
        {{$t("sureUnlock")}}
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
import UnLockIcon from "../../components/icons/unlock_icon.vue";
import DetailIcon from "../../components/icons/detail_icon.vue";
import Action from "./action.vue";
import Btoast from "../../components/btoast.vue";
import LoginModal from "../../components/loginModal.vue";
import TimeToolTips from "../../components/timeToolTips.vue";
import More from "../../components/more.vue";
import Copy from "../../components/copy.vue";
import Search from "./search.vue";
import Spinner from "../../components/spinner.vue";
import NoMessage from "../../components/noMessage.vue";

const [id,toastRef] = [ref("userToast"),ref(null)];
const [page,pageSize,total,cursor,logs] = [ref(1),ref(10),ref(0),ref(0),ref([])];

const [dataId] = [ref("")];
const [unlockLabel,showUnlockModal,deleteId] = [ref("unlockLabel"),ref("showUnlockModal"),ref("")];

const [noticeId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const loading = ref(false);

const [sequenceLockSearchKey,SequencePageKey] = [ref("sequenceLockSearch"),ref("SequencePage")];

const form = ref({
  orderKey:"",
  channelName:"",
  topicName:""
});

const filter = ((topic)=>{
  form.value.topicName = topic;
  search();
})

const search = (()=>{
  sessionStorage.setItem(sequenceLockSearchKey.value,JSON.stringify(form.value));
  //page.value = 1;
  sequenceLockLogs();
})

const maskString = ((id)=>{
  return Base.MaskString(id)
})

async function sequenceLockLogs() {
  logs.value = [];
  loading.value = true;
  try {
    let res = await sequenceLockApi.List(page.value,pageSize.value,form.value.orderKey,form.value.channelName,form.value.topicName);
    if(Object.keys(res).length > 0){
      logs.value.push(res);
    }

    setTimeout(()=>{
      loading.value = false;
    },800);
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
  if(sessionStorage.getItem(sequenceLockSearchKey.value)){
    form.value = JSON.parse(sessionStorage.getItem(sequenceLockSearchKey.value));
  }
  if(sessionStorage.getItem(SequencePageKey.value)){
    page.value = parseInt(sessionStorage.getItem(SequencePageKey.value));
  }

  sequenceLockLogs();
})

const [uRouter] = [useRouter()];
function detail(item){
  uRouter.push("/admin/log/dlq/detail/"+item._id);
}

function doUnlockModal(item){
  deleteId.value = "";
  dataId.value = "";
  const ele = document.getElementById("showUnlockModal");
  showUnlockModal.value = new bootstrap.Modal(ele);
  showUnlockModal.value.show(ele);
  deleteId.value = item.channel + ":" + item.topic + ":" + item.orderKey;
  dataId.value = "";
}

async function unlockInfo(){
  showUnlockModal.value.hide();
  if(deleteId.value === ""){
    toastRef.value.show("Missing Information");
    return;
  }
  try {
    let res = await sequenceLockApi.UnLock(deleteId.value);
    toastRef.value.show("success");
    await sequenceLockLogs();
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
  sessionStorage.setItem(SequencePageKey.value,page.value);

  sequenceLockLogs();
}

</script>
<style scoped>
.dlq{
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>