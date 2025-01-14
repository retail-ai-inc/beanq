<template>
  <div class="event">

    <div class="container-fluid">

      <!--search-->
      <Search :form="form" @search="search"/>
      <!--search end-->
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
                  <th scope="col">Action</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(item, key) in eventLogs" :key="key" style="height: 3rem;line-height:3rem">
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
                  <td>
                    <a class="btn btn-success icon-button" href="javascript:;" role="button" title="Retry" @click="retryModal(item)">
                      <RetryIcon />
                    </a>
                    <a class="btn btn-danger icon-button" href="javascript:;" role="button" title="Delete" @click="deleteModal(item)">
                      <DeleteIcon />
                    </a>
                    <a class="btn btn-primary icon-button" href="javascript:;" role="button" title="Edit" @click="editModal(item)">
                      <EditIcon />
                    </a>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>

      <!--edit modal-->
      <EditAction :label="infoDetailLabel" :id="showInfoDetail" :data="detail" @action="editInfo(detail)"></EditAction>
      <!--edit modal end-->
      <!--retry modal begin-->
      <Action :label="retryLabel" :id="showRetryModal" @action="retryInfo">
        <template #title="{title}">
          Are you sure to retry?
        </template>
        <template #body="{body}">
          Trying again will not restore the data
        </template>
      </Action>
      <!--retry modal end-->
      <!--delete modal begin-->
      <Action :label="deleteLabel" :id="showDeleteModal" @action="deleteInfo">
        <template #title="{title}">
          Are you sure to delete?
        </template>
        <template #body="{body}">
          If you need to restore, please contact the administrator.
        </template>
      </Action>
      <!--delete modal end-->
    </div>
  </div>
</template>
<script setup>
import { reactive,onMounted,toRefs,onUnmounted,computed } from "vue";
import { useRouter } from 'vueRouter';
import Pagination from "../components/pagination.vue";
import RetryIcon from "../components/icons/retry_icon.vue";
import DeleteIcon from "../components/icons/delete_icon.vue";
import EditIcon from "../components/icons/edit_icon.vue";
import Search from "./event/search.vue";
import EditAction from "../components/editAction.vue";
import Action from "../components/action.vue";

let data = reactive({
  eventLogs:[],
  page:1,
  pageSize:10,
  total:1,
  cursor:0,
  form:{
    id:"",
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


function deleteModal(item){
  data.deleteId = "";
  const ele = document.getElementById("showDeleteModal");
  data.deleteModal = new bootstrap.Modal(ele);
  data.deleteModal.show(ele);
  data.deleteId = item._id;
}

// delete log
async function deleteInfo(){

  if(data.deleteId == ""){
    return;
  }
  try {
    let res = await eventApi.Delete(data.deleteId);
  }catch (e) {
    console.log("delete err:",e);
  }

}

function retryModal(item){
  data.retryItem = {};
  const eleRetry = document.getElementById("showRetryModal");
  data.retryModal = new bootstrap.Modal(eleRetry);
  data.retryModal.show(eleRetry);
  data.retryItem = item;
}

// send payload into queue to consume it again
async function retryInfo(){

  if(data.retryItem._id == ""){
    return;
  }
  try{
    let res = await eventApi.Retry(data.retryItem._id,data.retryItem);
  }catch (e) {
    console.log("retry err:",e)
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
    if(res.code == "0000"){
      data.infoDetailModal.hide();
      return;
    }
  }catch (e) {
    console.log("edit err:",e);
  }

}

// search feature
async function search(){

  sessionStorage.setItem("id",data.form.id);
  sessionStorage.setItem("status",data.form.status);

  initEventSource();
}
// paging
async function changePage(page,cursor){
  data.page = page;
  data.cursor = cursor;
  sessionStorage.setItem("page",page)

  initEventSource();
}
const uRouter = useRouter();
function detailEvent(item){
  uRouter.push("detail/"+item.id);
}

function initEventSource(){

  if (data.sseEvent){
    data.sseEvent.close();
  }
  data.sseEvent = sseApi.Init(`event_log/list?page=${data.page}&pageSize=${data.pageSize}&id=${data.form.id}&status=${data.form.status}`);
  data.sseEvent.onopen = () =>{
    console.log("handshake success");
  }
  data.sseEvent.onerror = (err)=>{
    console.log("event err----",err);
  }
  data.sseEvent.addEventListener("event_log", function(res){
    let body =  JSON.parse(res.data);
    data.eventLogs = body.data.data;
    data.page =  body.data.cursor;
    data.total = Math.ceil(body.data.total / data.pageSize);
  })
}

onMounted(async()=>{

  data.form = {
    id:sessionStorage.getItem("id")??"",
    status:sessionStorage.getItem("status")??""
  };
  data.page = sessionStorage.getItem("page")??1;

  initEventSource();

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
.icon-button{
  width: 2.2rem;height:2.2rem;padding:0.2rem 0.5rem 0.5rem;margin-right: 0.2rem;
}
</style>