<template>
  <div class="event">

    <div class="container-fluid">

      <!--search-->
      <Search :form="form" @search="search"/>
      <!--search end-->
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
      <hr>
      <div class="row">
        <div class="col-12">
          <div class="table-responsive">
            <table class="table table-striped table-hover" style="table-layout: auto;">
              <thead>
                <tr>
                  <th scope="col" class="w-table-number">#</th>
                  <th scope="col" class="text-nowrap">Id</th>
                  <th scope="col" class="text-nowrap">Channel</th>
                  <th scope="col" class="text-nowrap">Topic</th>
                  <th scope="col" class="text-nowrap">Mood Type</th>
                  <th scope="col" class="text-center">Status</th>
                  <th scope="col" class="text-nowrap">Add Time</th>
                  <th scope="col" class="text-nowrap">Payload</th>
                  <th scope="col" class="text-center text-nowrap">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(item, key) in eventLogs" :key="key" style="height: 2rem;line-height:2rem">
                  <td class="text-right">
                    <router-link to="" class="nav-link text-primary" style="display: contents" v-on:click="detailEvent(item)">{{item._id}}</router-link>
                  </td>
                  <td class="">
                    {{item.id}}
                  </td>
                  <td>{{item.channel}}</td>
                  <td>{{item.topic}}</td>
                  <td>{{item.moodType}}</td>
                  <td class="text-center">
                    <span v-if="item.status == 'success'" class="text-success">{{item.status}}</span>
                    <span v-else-if="item.status == 'failed'" class="text-danger">{{item.status}}</span>
                    <span v-else-if="item.status == 'published'" class="text-warning">{{item.status}}</span>
                  </td>
                  <td>{{item.addTime}}</td>
                  <td>
                    <span class="d-block text-truncate" style="max-width: 30rem;">
                      <pre><code>{{item.payload}}</code></pre>
                    </span>
                  </td>
                  <td class="text-center text-nowrap">
                    <RetryIcon @action="retryModal(item)" style="margin: 0 .25rem"/>
                    <EditIcon @action="editModal(item)"/>
                    <DeleteIcon @action="deleteModal(item)" style="margin:0 .25rem;"/>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>

      <!--edit modal-->
      <EditAction :label="infoDetailLabel" :id="showInfoDetail" :data="detail" @action="editInfo"></EditAction>
      <!--edit modal end-->
      <!--retry modal begin-->
      <Action :label="retryLabel" :id="showRetryModal" :data-id="dataId" :warning="retryWarningHtml" :info="retryInfoHtml" @action="retryInfo">
        <template #title="{title}">
<!--          {{l.retryModal.title}}-->
        </template>
      </Action>
      <!--retry modal end-->
      <!--delete modal begin-->
      <Action :label="deleteLabel" :id="showDeleteModal" :data-id="dataId" @action="deleteInfo">
        <template #title="{title}">
<!--          {{l.deleteModal.title}}-->
        </template>
      </Action>
      <!--delete modal end-->
    </div>
    <Btoast :id="eventBtoastId" ref="eventRef"/>
  </div>
</template>
<script setup>
import { ref,inject,reactive,onMounted,toRefs,onUnmounted } from "vue";
import { useRouter,useRoute } from 'vueRouter';
import Pagination from "../../components/pagination.vue";
import RetryIcon from "../../components/icons/retry_icon.vue";
import DeleteIcon from "../../components/icons/delete_icon.vue";
import EditIcon from "../../components/icons/edit_icon.vue";
import Search from "./search.vue";
import EditAction from "./editAction.vue";
import Action from "../../components/action.vue";
import Btoast from "../../components/btoast.vue";

const l = ref(inject("i18n"));
const [eventBtoastId,eventRef] = [ref("eventBtoastId"),ref(null)];

let data = reactive({
  eventLogs:[],
  page:1,
  pageSize:10,
  total:1,
  cursor:0,
  form:{
    id:"",
    moodType:"",
    status:""
  },
  detail:{},
  isFormat:false,
  sseEvent:null,
  infoDetailModal:null,
  retryModal:null,
  deleteModal:null,
  retryItem:{},
  deleteId:"",
  infoDetailLabel:"infoDetailLabel",
  showInfoDetail:"showInfoDetail",
  retryLabel:"retryLabel",
  showRetryModal:"showRetryModal",
  deleteLabel:"deleteLabel",
  showDeleteModal:"showDeleteModal"
})

const [uRouter,route] = [useRouter(),useRoute()];
const [dataId] = [ref("")];
const [retryWarningHtml,retryInfoHtml] = [
    ref("Warning: Item retry cannot be undone!<br/> Please proceed with caution!"),
    ref("This operation will permanently retry the data of log.<br>\n" +
      "To prevent accidental actions, please confirm by entering the following:<br/>")
]

function deleteModal(item){
  data.deleteId = "";
  dataId.value = "";
  const ele = document.getElementById("showDeleteModal");
  data.deleteModal = new bootstrap.Modal(ele);
  data.deleteModal.show(ele);
  data.deleteId = item._id;
  dataId.value = item._id;
}

// delete log
async function deleteInfo(){

  if(data.deleteId === ""){
    eventRef.value.show("missing Id");
    return;
  }
  try {
    let res = await eventApi.Delete(data.deleteId);
    data.deleteModal.hide();
    eventRef.value.show(res.msg);

  }catch (e) {
    eventRef.value.show(e.error);
  }
}

function retryModal(item){
  data.retryItem = {};
  dataId.value = "";
  const eleRetry = document.getElementById("showRetryModal");
  data.retryModal = new bootstrap.Modal(eleRetry);
  data.retryModal.show(eleRetry);
  data.retryItem = item;
  dataId.value = item._id;
}

// send payload into queue to consume it again
async function retryInfo(){
  data.retryModal.hide();
  if(data.retryItem._id === ""){
    eventRef.value.show("missing Id");
    return;
  }
  try{
    let res = await eventApi.Retry(data.retryItem._id,data.retryItem);
    eventRef.value.show(res.msg);
  }catch (e) {
    eventRef.value.show(e.error);
  }

}
// trigger modal
function editModal(item){

  // sort keys
  data.detail = {
    _id:item._id,
    id:item.id,
    moodType:item.moodType,
    channel:item.channel,
    topic:item.topic,
    consumer:`${item.consumer}`,
    addTime:item.addTime,
    beginTime:item.beginTime,
    endTime:item.endTime,
    executeTime:item.executeTime,
    payload:item.payload,
    pendingRetry:item.pendingRetry,
    priority:item.priority,
    retry:item.retry,
    runTime:item.runTime,
    status:item.status
  };

  const ele = document.getElementById("showInfoDetail");
  data.infoDetailModal = new bootstrap.Modal(ele);
  data.infoDetailModal.show(ele);
}

async function editInfo(item){

  try{
    let res = await eventApi.Edit(item._id,item.payload);
    //if success
    eventRef.value.show(res.msg);
    data.infoDetailModal.hide();

  }catch (e) {
    eventRef.value.show(e.error);
  }

}

// search feature
async function search(){

  return uRouter.push(`/admin/log/event?id=${data.form.id}&status=${data.form.status}&moodType=${data.form.moodType}`).then(()=>{
    window.location.reload();
  });
}

// paging
async function changePage(page,cursor){
  data.page = page;
  data.cursor = cursor;
  sessionStorage.setItem("page",page)
  let apiUrl = `event_log/list?page=${data.page}&pageSize=${data.pageSize}&id=${data.form.id}&status=${data.form.status}&moodType=${data.form.moodType}`;
  initEventSource(apiUrl);
}

function detailEvent(item){
  uRouter.push("detail/"+item._id);
}

function initEventSource(apiUrl){

  if (data.sseEvent){
    data.sseEvent.close();
  }
  data.sseEvent = sseApi.Init(apiUrl);
  data.sseEvent.onopen = () =>{
    console.log("handshake success");
  }
  data.sseEvent.onerror = (err)=>{
    uRouter.replace("/login");
    console.log("event err----",err);
  }
  data.sseEvent.addEventListener("event_log", function(res){
    let body =  JSON.parse(res.data);
    data.eventLogs = body.data.data;
    data.page =  body.data.cursor;
    data.total = body.data.total;
  })
}

onMounted(async()=>{

  let [id,status,moodType] = [route.query.id,route.query.status,route.query.moodType];
  data.form = {
    id:id??"",
    status:status??"",
    moodType:moodType??""
  };
  data.page = sessionStorage.getItem("page")??1;

  let apiUrl = `event_log/list?page=${data.page}&pageSize=${data.pageSize}&id=${data.form.id}&status=${data.form.status}&moodType=${data.form.moodType}`;
  initEventSource(apiUrl);

})

onUnmounted(()=>{
  data.sseEvent.close();
})

const {eventLogs,form,page,total,cursor,detail,retryLabel,showRetryModal,deleteLabel,showDeleteModal,infoDetailLabel,showInfoDetail} = toRefs(data);

</script>
<style scoped>
.event{
  transition: opacity 0.5s ease;
  opacity: 1;
}
.table th, .table td {
  vertical-align: middle;
}
</style>