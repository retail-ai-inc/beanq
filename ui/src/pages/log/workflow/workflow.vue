<template>
  <div class="workflow">
    <div class="container-fluid">
      <div class="row mb-4">
        <div class="col">
          <h5 class="card-title">List of Workflow Log</h5>
        </div>
      </div>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
      <div class="row">
        <div class="col-12">
          <div v-if="workflowlogs.length <= 0" style="text-align: center;font-size: 1.2rem;">
            Hurrah! We processed all messages.
          </div>
          <div v-else class="table-responsive">
            <table class="table table-striped table-hover">
              <thead>
              <tr>
                <th scope="col">#</th>
                <th scope="col">GId</th>
                <th scope="col">TaskId</th>
                <th scope="col">Channel</th>
                <th scope="col">Topic</th>
                <th scope="col">Message Id</th>
                <th scope="col">Status</th>
                <th scope="col">Statement</th>
                <th scope="col">CreatedTime</th>
                <th scope="col">Action</th>
              </tr>
              </thead>
              <tbody>
              <tr v-for="(item, key) in workflowlogs" :key="key" style="height: 3rem;line-height:3rem">
                <th scope="row">{{item._id}}</th>
                <td><router-link to="" class="nav-link text-primary" style="display: contents">{{item.Gid}}</router-link></td>
                <td>{{item.TaskId}}</td>
                <td>{{item.Channel}}</td>
                <td>{{item.Topic}}</td>
                <td>{{item.MessageId}}</td>
                <td>
                  {{item.Status}}
                </td>
                <td>{{item.Statement}}</td>
                <td>{{item.CreatedAt}}</td>
                <td class="text-center text-nowrap">
                  <DeleteIcon @action="deleteModal(item)" style="margin:0 .25rem;"/>
                </td>
              </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>

    </div>
    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="deleteId" @action="deleteInfo">
      <template #title="{title}">
      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
  </div>
</template>
<script setup>
import { ref,onMounted,onUnmounted,computed } from "vue";
import Pagination from "../../components/pagination.vue";
import DeleteIcon from "../../components/icons/delete_icon.vue";
import Action from "../../components/action.vue";
import Btoast from "../../components/btoast.vue";

const [id,toastRef] = [ref("userToast"),ref(null)];
const [workflowlogs,page,pageSize,total,cursor] = [ref([]),ref(1),ref(10),ref(0),ref(0)];
const [deleteLabel,showDeleteModal,deleteId] = [ref("deleteLabel"),ref("showDeleteModal"),ref("")]
// logs
const getWorkFLowLogs=(async (pageV,pageSizeV)=>{
  let res = await workflowApi.List(pageV,pageSizeV);
  const {code,msg,data} = res;
  if(code !== "0000"){
    toastRef.value.show(msg);
    return;
  }
  workflowlogs.value = data.data;
  total.value = data.total;
  page.value =  data.cursor;
  cursor.value = data.cursor;
})

// paging
function changePage(pageVal,cursorVal){
  page.value = pageVal;
  cursor.value = cursorVal;
  sessionStorage.setItem("page",pageVal);
  getWorkFLowLogs(page.value,pageSize.value);
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
    const {code,msg,data} = res;
    data.deleteModal.hide();
    toastRef.value.show(msg);
    if(code === "0000"){
      await getWorkFLowLogs(page.value,pageSize.value);
    }
  }catch (e) {
    toastRef.value.show(e.error);
  }

}

onMounted(()=>{
  page.value = sessionStorage.getItem("page")??1;
  getWorkFLowLogs(page.value,pageSize.value);

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