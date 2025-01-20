<template>
  <div class="table-responsive opt-log">
    <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
    <table class="table table-striped table-hover">
      <thead>
      <tr>
        <th scope="col">#</th>
        <th scope="col">Add Time</th>
        <th scope="col">Account</th>
        <th scope="col">Visit</th>
        <th scope="col">Data</th>
        <th scope="col">Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item, key) in list" :key="key" style="height: 2.5rem;">
        <th scope="row" style="width: 5%">{{parseInt(key)+1}}</th>
        <td>
            <pre><code>{{item.addTime}}</code></pre>
        </td>
        <td style="width: 15%">{{item.user}}</td>
        <td style="width: 30%">
          <span class="d-inline-block text-truncate" style="max-width: 50rem">
            <pre><code>{{item.uri}}</code></pre>
          </span>
        </td>
        <td style="width: 45%">
          <span class="d-inline-block text-truncate" style="max-width: 400px">
            {{item.data}}
          </span>
        </td>
        <td style="width: 5%">
            <Delete_icon @action="deleteShowModal(item)"/>
        </td>
      </tr>
      </tbody>
    </table>
    <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
    <Btoast :id="id" ref="toastRef"/>
    <Action :label="deleteLabel" :id="showDeleteModal" @action="deleteLog">
      <template #title="{title}">
        Are you sure to delete?
      </template>
      <template #body="{body}">
        If you need to restore, please contact the administrator.
      </template>
    </Action>
  </div>
</template>
<script setup>
import { ref,onMounted,onUnmounted } from "vue";
import Delete_icon from "../components/icons/delete_icon.vue";
import Btoast from "../components/btoast.vue";
import Action from "../components/action.vue";
import Pagination from "../components/pagination.vue";

const [list,id,toastRef] = [ref([]),ref("optLog"),ref(null)];
const [deleteLabel,showDeleteModal,deleteModal,mid] = [ref("deleteLabel"),ref("showDeleteModal"),ref(null),ref("")];
const [page,pageSize,total,cursor] = [ref(1),ref(10),ref(0),ref(1)];

onMounted(async ()=>{

  let res = await logApi.OptLog(page.value,pageSize.value);
  const {data} = res;

  list.value = data.data;
  total.value = Math.ceil(data.total / pageSize.value);
  cursor.value = data.cursor;
})

onUnmounted(()=>{
})

function deleteShowModal(item){
  const ele = document.getElementById("showDeleteModal");
  deleteModal.value = new bootstrap.Modal(ele);
  deleteModal.value.show(ele);
  mid.value = item._id;
}

async function deleteLog(){

  try {
    let res = await logApi.DeleteOptLog(mid.value);
    if(res.code == "0000"){
      let res = await logApi.OptLog(page.value,pageSize.value);
      const {data} = res;
      list.value = data.data;
      total.value =  Math.ceil(data.total / pageSize.value);
      cursor.value = data.cursor;
      deleteModal.value.hide();
    }
  }catch (e) {
    toastRef.value.show(e.message);
  }

}

async function changePage(pageo,cursoro){

  page.value = pageo;
  let res = await logApi.OptLog(page.value,pageSize.value);
  const {data} = res;
  list.value = data.data;
  total.value = Math.ceil(data.total / pageSize.value);
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