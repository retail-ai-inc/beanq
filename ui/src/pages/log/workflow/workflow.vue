<template>
  <div class="workflow">
    <div class="container-fluid">
      <div class="row mb-4">
        <div class="col">
          <h5 class="card-title">List of Workflow Log</h5>
        </div>
      </div>

      <Search :form="form" @search="search"/>
      <Spinner v-if="loading" style="margin: 1rem 0"/>
      <div v-else>
        <NoMessage v-if="workflowlogs.length <= 0" style="margin:1rem 0"/>
        <div v-else>
          <div class="d-flex flex-row justify-content-end">
            <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
            <select class="form-select form-select-sm" aria-label=".form-select-sm example" style="height:35px;width:8%;margin-left: 10px;" @change="changeItem">
              <option :selected="pageSize===10" value="10">10 / page</option>
              <option value="20"  :selected="pageSize === 20" >20 / page</option>
              <option value="50" :selected="pageSize===50">50 / page</option>
              <option value="100" :selected="pageSize===100">100 / page</option>
            </select>
          </div>
          <div class="row">
            <div class="col-12">

              <div class="table-responsive">
                <table class="table table-striped table-hover">
                  <thead>
                  <tr>
                    <th scope="col">#</th>
                    <th scope="col">GId</th>
                    <th scope="col">Task Id</th>
                    <th scope="col">Channel</th>
                    <th scope="col">Topic</th>
                    <th scope="col">Status</th>
                    <th scope="col">Option</th>
                    <th scope="col">Statement</th>
                    <th scope="col">Error</th>
                    <th scope="col">Created AT</th>
                    <th scope="col">Updated AT</th>
                    <th scope="col">Action</th>
                  </tr>
                  </thead>
                  <tbody>
                  <tr v-for="(item, key) in workflowlogs" :key="key" style="height: 3rem;line-height:3rem">
                    <td>{{item.auto_id}}</td>
                    <td><router-link to="" class="nav-link text-primary" style="display: contents">{{item.Gid}}</router-link></td>
                    <td>{{item.TaskId}}</td>
                    <td>{{item.Channel}}</td>
                    <td>{{item.Topic}}</td>
                    <td>
                      {{item.Status}}
                    </td>
                    <td>
                      {{item.Option}}
                    </td>
                    <td>{{item.Statement}}</td>
                    <td>{{item.Error}}</td>
                    <td>{{item.CreatedAt}}</td>
                    <td>{{item.UpdatedAT}}</td>
                    <td class="text-center text-nowrap">
                      <DeleteIcon @action="deleteModal(item)" style="margin:0 .25rem;"/>
                    </td>
                  </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
          <div class="d-flex flex-row justify-content-end">
            <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
            <select class="form-select form-select-sm" aria-label=".form-select-sm example" style="height:35px;width:8%;margin-left: 10px;" @change="changeItem">
              <option :selected="pageSize===10" value="10">10 / page</option>
              <option value="20"  :selected="pageSize === 20" >20 / page</option>
              <option value="50" :selected="pageSize===50">50 / page</option>
              <option value="100" :selected="pageSize===100">100 / page</option>
            </select>
          </div>
        </div>
      </div>
    </div>
    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="deleteId" @action="deleteInfo">
      <template #title="{title}">
      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
    <LoginModal :id="noticeId" ref="loginModal"/>
  </div>
</template>
<script setup>
import { ref,onMounted,onUnmounted,computed } from "vue";
import Pagination from "../../components/pagination.vue";
import DeleteIcon from "../../components/icons/delete_icon.vue";
import Action from "../../components/action.vue";
import Btoast from "../../components/btoast.vue";
import LoginModal from "../../components/loginModal.vue";
import Spinner from "../../components/spinner.vue";
import NoMessage from "../../components/noMessage.vue";
import Search from "./search.vue";

const [id,toastRef] = [ref("userToast"),ref(null)];
const [workflowlogs,page,pageSize,total,cursor] = [ref([]),ref(1),ref(10),ref(0),ref(0)];
const [deleteLabel,showDeleteModal,deleteId] = [ref("deleteLabel"),ref("showDeleteModal"),ref("")];
const [noticeId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const loading = ref(false);
// logs
const getWorkFLowLogs=(async (pageV,pageSizeV,channelName,topicName,status)=>{
  loading.value = true;
  try {
    let res = await workflowApi.List(pageV,pageSizeV,channelName,topicName,status);

    let ndata = res.data || [];

    ndata = ndata.map((item,index)=>{
      return {
        ...item,
        auto_id:(pageV -1 ) * pageSizeV + index + 1
      }
    })

    workflowlogs.value = ndata;
    total.value = res.total;
    page.value =  res.cursor;
    cursor.value = res.cursor;
    setTimeout(()=>{
      loading.value = false;
    },800)
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    toastRef.value.show(err);
  }

})

const form = ref({
  channelName:"",
  topicName:"",
  status:""
})
const search = async ()=>{
  return await getWorkFLowLogs(page.value,pageSize.value,form.value.channelName,form.value.topicName,form.value.status);
}

function changeItem(e){
  pageSize.value = parseInt(e.target.value);
  Storage.SetItem("pageSize",pageSize.value);
  getWorkFLowLogs(page.value,pageSize.value,form.value.channelName,form.value.topicName,form.value.status);
}

// paging
function changePage(pageVal,cursorVal){
  page.value = pageVal;
  cursor.value = cursorVal;
  Storage.SetItem("page",pageVal);
  getWorkFLowLogs(page.value,pageSize.value,form.value.channelName,form.value.topicName,form.value.status);
}

function deleteModal(item){
  deleteId.value = "";
  const ele = document.getElementById("showDeleteModal");
  deleteModal.value = new bootstrap.Modal(ele);
  deleteModal.value.show(ele);
  deleteId.value = item._id;
}

async function deleteInfo(){
  deleteModal.value.hide();
  if(deleteId.value === ""){
    toastRef.value.show("missing Id");
    return;
  }
  try {
    let res = await workflowApi.Delete(deleteId.value);

    deleteModal.value.hide();
    toastRef.value.show("success");
    await getWorkFLowLogs(page.value,pageSize.value,form.value.channelName,form.value.topicName,form.value.status);

  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    toastRef.value.show(err.error);
  }

}

onMounted(()=>{
  page.value = Storage.GetItem("page")??1;
  pageSize.value = parseInt(Storage.GetItem("pageSize"))??10;

  getWorkFLowLogs(page.value,pageSize.value,form.value.channelName,form.value.topicName,form.value.status);

})

onUnmounted(()=>{
  Storage.SetItem("page",1);
  Storage.SetItem("pageSize",10);
})

</script>
<style scoped>
.workflow{
  transition: opacity 0.5s ease;
  opacity: 1;
}
</style>