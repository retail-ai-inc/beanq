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
          <button type="button" class="btn btn-primary" @click="addUserModal">Add</button>
        </div>
    </div>

    <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
    <table class="table table-striped table-hover" style="table-layout: auto;">
      <thead>
      <tr>
        <th scope="col" class="w-table-number">#</th>
        <th scope="col" class="text-nowrap">Account</th>
        <th scope="col" class="text-nowrap">Active</th>
        <th scope="col" class="text-nowrap">Type</th>
        <th scope="col" class="text-nowrap">Detail</th>
        <th scope="col" class="text-center">Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item, key) in users" :key="key" style="height: 3rem;line-height:3rem">
        <td class="text-right">{{item._id}}</td>
        <td>{{item.account}}</td>
        <td>
          <span :class="item.active == 1 ? 'green' : 'red'">{{item.active == "1" ? "active" :"locked"}}</span>
        </td>
        <td>{{item.type}}</td>
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
    <div class="modal fade" id="addUserDetail" data-bs-keyboard="false" tabindex="-1" aria-labelledby="addUserDetailLabel">
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h1 class="modal-title fs-5" id="addUserDetailLabel">
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
                  v-model="roleForm.account"
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
              <tree :nodes="nodes" :checkedIds="ids" @choose="chooseNode"/>
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

    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="userId" @action="deleteRole">
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
const [users,accountReadOnly,addUserDetail] = [ref([]),ref(false),ref(null)];
const [page,pageSize,cursor,total] = [ref(1),ref(10),ref(0),ref(0)];
const [nameInput,userId] = [ref(""),ref("")];
const roleForm = ref({name:"",roles:[]});
const nodes = ref(role);


async function userList(){
  let res = await userApi.List(page.value,pageSize.value,nameInput.value);
  const {code,msg,data} = res;
  if(code !== "0000"){

  }
  users.value = data.data;
  cursor.value = data.cursor;
  total.value = data.total ;
}

onMounted( ()=>{

  userList();
  const ele = document.getElementById("addUserDetail");
  ele.addEventListener('hidden.bs.modal', () => {
    roleForm.value = {};
    accountReadOnly.value = false;
  });

});

onUnmounted(()=>{
  const ele = document.getElementById('addUserDetail');
  if (ele) {
    ele.removeEventListener('hidden.bs.modal', () => {

    });
  }
});

const ids = ref([]);
function tileTree(tree) {
  return _.flatMap(tree,(node)=>{
    let children = node.children ? tileTree(node.children) : [];
    return [node,...children];
  });
}

const tileNodes = computed(()=>{
  return tileTree(role);
})

function getParent(id,trees){
  let node = _.find(trees,function (v){
    return v.id === id;
  })
  if(node !== undefined){
    ids.value.push(node.id);
  }

  if(node !== undefined && (('pid' in node) && node.pid > 0)){
    getParent(node.pid,trees);
  }
}

function chooseNode(event){

  let id = event.target.getAttribute("id");
  let isChecked = event.target.checked;
  let trees = tileNodes.value;
  getParent(parseInt(id),trees);
  ids.value = _.uniq(ids.value);
  if(isChecked === false){
    _.pull(ids.value,parseInt(id));
  }
}

function addRole(){

}

function editRole(){

}

function deleteRole(){

}

function SearchByAccount(){
  userList();
}

function changePage(page,cursor){
  page.value = page;
  cursor.value = cursor;
  sessionStorage.setItem("page",page);

  userList();
}

function checkValid(e){

  let next = e.currentTarget.nextElementSibling;
  next.style.display = "none";
  //check account
  if(e.currentTarget.id === "nameInput"){
    next.innerHTML = "Please input an Email account";
    if(roleForm.value.name === "" || Base.CheckEmail(roleForm.value.name) === false){
      next.style.display = "block";
    }
  }
}

function addUserModal(){
  const ele = document.getElementById("addUserDetail");
  addUserDetail.value = new bootstrap.Modal(ele);
  addUserDetail.value.show(ele);
}

async function addUser(e){

  try {
    let next = e.currentTarget.nextElementSibling;
    let res = await userApi.Add(roleForm.value);
    if(res.code != "0000"){
      next.style.display = "block";
      next.innerHTML = res.msg;
      return
    }
    next.style.display = "none";
    addUserDetail.value.hide();
    await userList();
  }catch (e) {
    toastRef.value.show(e.message);
  }

}

async function editUser(){
  let res = await userApi.Edit(roleForm.value);
  addUserDetail.value.hide();
  await userList();

  return
}

function deleteUserModal(item){
  account.value = "";
  userId.value = "";
  const ele = document.getElementById("showDeleteModal");
  delModal.value = new bootstrap.Modal(ele);
  delModal.value.show(ele);
  account.value  = item._id;
  userId.value = item._id;
}

async function deleteUser(){
  delModal.value.hide();

  try {
    let res = await userApi.Delete(account.value);
    if(res.code == "0000"){
      await userList();

      toastRef.value.show("Success");
    }
  }catch (e) {
    toastRef.value.show(e.message);
  }

}

function editUserModal(item){
  roleForm.value = item;
  accountReadOnly.value = true;
  const ele = document.getElementById("addUserDetail");

  addUserDetail.value = new bootstrap.Modal(ele);
  addUserDetail.value.show(ele);

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