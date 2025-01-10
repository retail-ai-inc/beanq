<template>
  <div class="event">

    <div class="container-fluid">

      <!--search-->
        <div class="row">

          <div class="col-md-2">
            <div class="row">
            <label for="formId" class="col-md-4 col-form-label text-end">Id:</label>
            <div class="col-md-8">
              <input type="text" class="form-control" id="formId" name="formId"  v-model="form.id">
            </div>
            </div>
          </div>

          <div class="col-md-3">
            <div class="row">
              <label for="formStatus" class="col-md-4 col-form-label text-end">Status:</label>
              <div class="col-md-8">
                <select class="form-select" aria-label="Default select" id="formStatus" name="formStatus" style="cursor: pointer" v-model="form.status">
                  <option selected value="">Open this select</option>
                  <option value="published">Published</option>
                  <option value="success">Success</option>
                  <option value="failed">Failed</option>
                </select>
              </div>
            </div>
          </div>

          <div class="col-2">
            <div class="col-auto">
              <button type="submit" class="btn btn-primary mb-3" @click="search">Search</button>
            </div>
          </div>
        </div>
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
                <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-collection-play" viewBox="0 0 16 16">
                  <path d="M2 3a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 0-1h-11A.5.5 0 0 0 2 3zm2-2a.5.5 0 0 0 .5.5h7a.5.5 0 0 0 0-1h-7A.5.5 0 0 0 4 1zm2.765 5.576A.5.5 0 0 0 6 7v5a.5.5 0 0 0 .765.424l4-2.5a.5.5 0 0 0 0-.848l-4-2.5z"/>
                  <path d="M1.5 14.5A1.5 1.5 0 0 1 0 13V6a1.5 1.5 0 0 1 1.5-1.5h13A1.5 1.5 0 0 1 16 6v7a1.5 1.5 0 0 1-1.5 1.5h-13zm13-1a.5.5 0 0 0 .5-.5V6a.5.5 0 0 0-.5-.5h-13A.5.5 0 0 0 1 6v7a.5.5 0 0 0 .5.5h13z"/>
                </svg>
              </a>
              <a class="btn btn-danger icon-button" href="javascript:;" role="button" title="Delete" @click="deleteModal(item)">
                <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-trash" viewBox="0 0 16 16">
                  <path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/>
                  <path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/>
                </svg>
              </a>
              <a class="btn btn-primary icon-button" href="javascript:;" role="button" title="Edit" @click="editModal(item)">
                <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-pencil-square" viewBox="0 0 16 16">
                  <path d="M15.502 1.94a.5.5 0 0 1 0 .706L14.459 3.69l-2-2L13.502.646a.5.5 0 0 1 .707 0l1.293 1.293zm-1.75 2.456-2-2L4.939 9.21a.5.5 0 0 0-.121.196l-.805 2.414a.25.25 0 0 0 .316.316l2.414-.805a.5.5 0 0 0 .196-.12l6.813-6.814z"/>
                  <path fill-rule="evenodd" d="M1 13.5A1.5 1.5 0 0 0 2.5 15h11a1.5 1.5 0 0 0 1.5-1.5v-6a.5.5 0 0 0-1 0v6a.5.5 0 0 1-.5.5h-11a.5.5 0 0 1-.5-.5v-11a.5.5 0 0 1 .5-.5H9a.5.5 0 0 0 0-1H2.5A1.5 1.5 0 0 0 1 2.5v11z"/>
                </svg>
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
      <div class="modal fade" id="infoDetail" data-bs-keyboard="false" tabindex="-1" aria-labelledby="infoDetailLabel" aria-hidden="true">
        <div class="modal-dialog modal-lg">
          <div class="modal-content">
            <div class="modal-header">
              <h1 class="modal-title fs-5" id="infoDetailLabel">Edit Payload</h1>
              <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
              <div class="mb-3 row" v-for="(item,key) in detail" :key="key">
                <label :for="key" class="col-sm-2 col-form-label" style="font-weight: bold">{{key}}</label>
                <div class="col-sm-10">

                  <div id="payloadAlertInfo" v-if="key === 'payload'">
                  </div>

                  <textarea class="form-control" id="payload" rows="3" v-if="key === 'payload'" v-model="detail.payload" @blur="payloadTrigger"></textarea>
                  <input type="text" readonly :id="key" class="form-control-plaintext" :value="item" v-else>
                </div>
              </div>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
              <button type="button" class="btn btn-primary" @click="editInfo(detail)">Edit</button>
            </div>
          </div>
        </div>
      </div>
      <!--edit modal end-->
      <!--retry modal begin-->
      <div class="modal fade" data-bs-keyboard="false" tabindex="-1" aria-labelledby="retryLabel" aria-hidden="true" id="retryModal">
        <div class="modal-dialog modal-md modal-dialog-centered">
          <div class="modal-content">
            <div class="modal-header">
              <h1 class="modal-title fs-5" id="retryLabel">Are you sure to retry?</h1>
              <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
              <p>Trying again will not restore the data</p>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-light" data-bs-dismiss="modal">Cancel</button>
              <button type="button" class="btn btn-danger" @click="retryInfo">Yes</button>
            </div>
          </div>
        </div>
      </div>
      <!--retry modal end-->
      <!--delete modal begin-->
      <div class="modal fade" data-bs-keyboard="false" tabindex="-1" aria-labelledby="deleteLabel" aria-hidden="true" id="deleteModal">
        <div class="modal-dialog modal-md modal-dialog-centered">
          <div class="modal-content">
            <div class="modal-header">
              <h1 class="modal-title fs-5" id="deleteLabel">Are you sure to delete?</h1>
              <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
              <p>If you need to restore, please contact the administrator.</p>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-light" data-bs-dismiss="modal">Cancel</button>
              <button type="button" class="btn btn-danger" @click="deleteInfo">Yes</button>
            </div>
          </div>
        </div>
      </div>
      <!--delete modal end-->
    </div>
  </div>
</template>
<script setup>
import { reactive,onMounted,toRefs,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';
import Pagination from "../components/pagination.vue";

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
  deleteId:""
})


function deleteModal(item){
  data.deleteId = "";
  const ele = document.getElementById("deleteModal");
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
  const eleRetry = document.getElementById("retryModal");
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
  const ele = document.getElementById("infoDetail");
  data.infoDetailModal = new bootstrap.Modal(ele);
  data.infoDetailModal.show(ele);
}

// Verify the JSON format of the payload
async function payloadTrigger(){

  data.isFormat = false;
  try {
    await JSON.parse(data.detail.payload);
  }catch (e) {
    data.isFormat = true;
  }
  if (data.isFormat === true){
      await eventApi.Alert("Must be in JSON format","danger");
      return;
  }
  const alertTrigger = new bootstrap.Alert('#my-alert');
  alertTrigger.close();

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

const {eventLogs,form,page,total,cursor,detail} = toRefs(data);

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