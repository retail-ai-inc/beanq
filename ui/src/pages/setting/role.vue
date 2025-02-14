<template>
  <div class="role">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of Admin Roles</h5>
      </div>
    </div>
    <div class="form-row mb-3">
        <div class="col">
          <input type="text" class="form-control" id="formId" name="formId" v-model="nameInput" placeholder="Search by role name">
        </div>
        <div class="col-auto" style="margin:0 .75rem;">
          <button type="submit" class="btn btn-primary" @click="SearchByAccount">Search</button>
        </div>
        <div class="col-auto border-left" style="padding-left: .85rem">
          <button type="button" class="btn btn-primary" @click="addRoleModal">Add</button>
        </div>
    </div>

    <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
    <table class="table table-striped table-hover" style="table-layout: auto;">
      <thead>
      <tr>
        <th scope="col" class="w-table-number">#_ID</th>
        <th scope="col" class="text-nowrap">Name</th>
        <th scope="col" class="text-nowrap">Detail</th>
        <th scope="col" class="text-center">Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item, key) in users" :key="key" style="height: 3rem;line-height:3rem">
        <td class="text-right">{{item._id}}</td>
        <td>{{item.name}}</td>
        <td>
          <span class="d-inline-block text-truncate" style="max-width: 5rem;">
            {{item.detail}}
          </span>
        </td>
        <td class="text-center text-nowrap">
          <EditIcon @action="editUserModal(item)" />
          <DeleteIcon @action="deleteUserModal(item)" style="margin:0 .25rem;" />
        </td>
      </tr>
      </tbody>

    </table>
    <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>

    <!--add user modal-->
    <div class="modal fade" id="addRoleDetail" data-bs-keyboard="false" tabindex="-1" aria-labelledby="addRoleDetailLabel">
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h1 class="modal-title fs-5" id="addRoleDetailLabel">
              {{accountReadOnly == true ? "Edit Role" : "Add Role"}}
            </h1>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
          </div>
          <div class="modal-body">
            <div class="mb-3">
              <label for="nameInput" class="form-label">Role Name
              </label>
              <input
                  type="text"
                  class="form-control"
                  id="nameInput"
                  @blur="checkValid"
                  v-model="roleForm.name"
                  :readonly="accountReadOnly == true ? 'readonly': false"
                  :disabled="accountReadOnly === true ? 'disabled': false"
                  placeholder="Please input a role name"
              />
              <div class="invalid-feedback">
                Please input a role name.
              </div>
            </div>
            <div class="mb-3">
              <label class="form-label">Roles</label>
              <tree :nodes="nodes" :checkedIds="roleForm.roles" @choose="chooseNode"/>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">{{l.closeButton}}</button>
            <button type="button" class="btn btn-primary" @click="addRole" v-if="accountReadOnly == false">{{l.addButton}}</button>
            <button type="button" class="btn btn-primary" @click="editRole" v-else>{{l.editButton}}</button>
            <div class="invalid-feedback">
            </div>
          </div>
        </div>
      </div>
    </div>
    <!--add user modal end-->

    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="roleId" @action="deleteRole">
      <template #title="{title}">
        {{l.deleteModal.title}}
      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
  </div>
</template>
<script setup>
import { ref,inject,onMounted,onUnmounted,computed } from "vue";
import DeleteIcon from "../components/icons/delete_icon.vue";
import EditIcon from "../components/icons/edit_icon.vue";
import Pagination from "../components/pagination.vue";
import Action from "../components/action.vue";
import Btoast from "../components/btoast.vue";
import Tree from "../components/tree.vue";

const l = ref(inject("i18n"));

const [deleteLabel,delModal,showDeleteModal,account] = [ref("deleteLabel"),ref(null),ref("showDeleteModal"),ref("")];
const [id,toastRef] = [ref("userToast"),ref(null)];
const [users,accountReadOnly,addRoleDetail] = [ref([]),ref(false),ref(null)];
const [page,pageSize,cursor,total] = [ref(1),ref(10),ref(0),ref(0)];
const [nameInput,roleId] = [ref(""),ref("")];
const roleForm = ref({name:"",roles:[]});
const nodes = ref(role);


async function roleList(){
  let res = await roleApi.List(page.value,pageSize.value,nameInput.value);
  const {code,msg,data} = res;
  if(code !== "0000"){

  }
  users.value = data.data;
  cursor.value = data.cursor;
  total.value = data.total ;
}

onMounted( ()=>{
  roleList();
  const ele = document.getElementById("addRoleDetail");
  ele.addEventListener('hidden.bs.modal', () => {
    roleForm.value = {};
    accountReadOnly.value = false;
  });

});
//
// onUnmounted(()=>{
//   const ele = document.getElementById('addRoleDetail');
//   if (ele) {
//     ele.removeEventListener('hidden.bs.modal', () => {
//
//     });
//   }
// });

function tileTree(tree) {
  return _.flatMap(tree,(node)=>{
    let children = node.children ? tileTree(node.children) : [];
    return [node,...children];
  });
}

const tileNodes = computed(()=>{
  return tileTree(role);
})

let ids = ref([]);
function getC(id){
    let a = _.filter(tileNodes.value,function (v) {
      return v.pid === id;
    })
    for(let i=0;i<a.length;i++){
      ids.value.push(a[i].id);
      getC(a[i].id);
    }
}
function getP(pid){
  let a = _.filter(tileNodes.value,function (v) {
    return v.id === pid;
  })
  for(let i=0;i<a.length;i++){
    ids.value.push(a[i].id);
    getP(a[i].pid);
  }
}

function chooseNode(event){

  let id = event.target.getAttribute("id");
  let isChecked = event.target.checked;

  ids.value = [];
  getP(parseInt(id));
  getC(parseInt(id));
  if(isChecked === false){
    roleForm.value.roles = _.difference(roleForm.value.roles,parseInt(id));
  }else{
    roleForm.value.roles.push(...ids.value);
  }

}

function SearchByAccount(){
  roleList();
}

function changePage(page,cursor){
  page.value = page;
  cursor.value = cursor;
  sessionStorage.setItem("page",page);

  roleList();
}

function checkValid(e){

  let next = e.currentTarget.nextElementSibling;
  next.style.display = "none";
  //check account
  if(e.currentTarget.id === "nameInput"){
    next.innerHTML = "Please input a name";
    if(roleForm.value.name === ""){
      next.style.display = "block";
    }
  }
}

function addRoleModal(){
  const ele = document.getElementById("addRoleDetail");
  addRoleDetail.value = new bootstrap.Modal(ele);
  addRoleDetail.value.show(ele);
}

async function addRole(e){

  try {
    let next = e.currentTarget.nextElementSibling;
    let res = await roleApi.Add(roleForm.value);
    if(res.code != "0000"){
      next.style.display = "block";
      next.innerHTML = res.msg;
      return
    }
    next.style.display = "none";
    addRoleDetail.value.hide();
    await roleList();
  }catch (e) {
    toastRef.value.show(e.message);
  }

}

async function editRole(){

  let res = await roleApi.Edit(roleId.value,roleForm.value);
  addRoleDetail.value.hide();
  await roleList();

  return
}

function deleteUserModal(item){
  account.value = "";
  roleId.value = "";
  const ele = document.getElementById("showDeleteModal");
  delModal.value = new bootstrap.Modal(ele);
  delModal.value.show(ele);
  account.value  = item._id;
  roleId.value = item._id;
}

async function deleteRole(){
  delModal.value.hide();
  try {
    let res = await roleApi.Delete(roleId.value);
    if(res.code == "0000"){
      await roleList();

      toastRef.value.show("Success");
    }
  }catch (e) {
    toastRef.value.show(e.message);
  }

}

function editUserModal(item){
  roleForm.value = item;
  accountReadOnly.value = true;
  roleId.value = item._id;
  const ele = document.getElementById("addRoleDetail");

  addRoleDetail.value = new bootstrap.Modal(ele);
  addRoleDetail.value.show(ele);

}

</script>

<style scoped>
.role{
  transition: opacity 0.5s ease;
  opacity: 1;
}

.green{
  color:var(--bs-success);
}
.red{
  color:var(--bs-danger);
}
.form-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}
.border-left {
  border-left: 1px solid #dee2e6 !important;
}
.form-label {
  font-weight: bold;
}
</style>