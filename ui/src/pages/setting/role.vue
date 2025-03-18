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
          <button type="button" class="btn btn-primary" @click="addRoleModal">{{$t('add')}}</button>
        </div>
    </div>

    <div class="text-center" v-if="loading">
      <div class="spinner-border" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>
    <div v-else>
      <div v-if="roles.length <= 0" style="text-align: center">
        create some admin ,please click the <button type="button" class="btn btn-primary" @click="addRoleModal">{{$t('add')}}</button>
      </div>
      <div v-else>
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
          <tr v-for="(item, key) in roles" :key="key" style="height: 3rem;line-height:3rem">
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
      </div>
    </div>

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
            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">{{$t('close')}}</button>
            <button type="button" class="btn btn-primary" @click="addRole" v-if="accountReadOnly == false">{{$t('add')}}</button>
            <button type="button" class="btn btn-primary" @click="editRole" v-else>{{$t('edit')}}</button>
            <div class="invalid-feedback">
            </div>
          </div>
        </div>
      </div>
    </div>
    <!--add user modal end-->

    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="roleId" :warning="$t('retryWarningHtml')" :info="$t('retryInfoHtml')" @action="deleteRole">
      <template #title="{title}">

      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
    <LoginModal :id="loginId" ref="loginModal"/>
  </div>
</template>
<script setup>
import { ref,onMounted,watch,computed } from "vue";
import DeleteIcon from "../components/icons/delete_icon.vue";
import EditIcon from "../components/icons/edit_icon.vue";
import Pagination from "../components/pagination.vue";
import Action from "../components/action.vue";
import Btoast from "../components/btoast.vue";
import Tree from "../components/tree.vue";
import LoginModal from "../components/loginModal.vue";


const nav = computed(()=>{
  return Nav;
})

const [deleteLabel,delModal,showDeleteModal,account] = [ref("deleteLabel"),ref(null),ref("showDeleteModal"),ref("")];
const [id,toastRef] = [ref("userToast"),ref(null)];
const [roles,accountReadOnly,addRoleDetail] = [ref([]),ref(false),ref(null)];
const [page,pageSize,cursor,total] = [ref(1),ref(10),ref(0),ref(0)];
const [nameInput,roleId] = [ref(""),ref("")];
const roleForm = ref({name:"",roles:[]});
const nodes = ref(Nav);
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const loading = ref(false);


async function roleList(){
  loading.value = true;
  try {
    let res = await roleApi.List(page.value,pageSize.value,nameInput.value);

    roles.value = res.data;
    cursor.value = res.cursor;
    total.value = res.total ;
    setTimeout(()=>{
      loading.value = false;
    },800)
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e);
  }

}

onMounted( ()=>{
  roleList();
  const ele = document.getElementById("addRoleDetail");
  ele.addEventListener('hidden.bs.modal', () => {
    roleForm.value = {};
    accountReadOnly.value = false;
  });

});

const tileNodes = computed(()=>{
  return roleApi.TileTree(nodes.value);
})

const ids = ref([]);
function getChild(id){
    let a = _.filter(tileNodes.value,function (v) {
      return v.pid === id;
    })
    for(let i=0;i<a.length;i++){
      ids.value.push(a[i].id);
      getChild(a[i].id);
    }
}
function getParent(pid){
  let a = _.filter(tileNodes.value,function (v) {
    return v.id === pid;
  })
  for(let i=0;i<a.length;i++){
    ids.value.push(a[i].id);
    getParent(a[i].pid);
  }
}
const lastIds = ref([]); // ids result,from last checked
function chooseNode(event){

  let id = event.target.getAttribute("id");
  let isChecked = event.target.checked;

  let obj = _.find(tileNodes.value,function (v) {
    return v.id === parseInt(id);
  })

  ids.value = [];
  getParent(parseInt(id));
  getChild(parseInt(id));

  if(isChecked === false){
    //cancel  checked
    let lids = lastIds.value;

    if(!_.has(obj,"children")){
      _.remove(lids,(x)=> x === parseInt(id));
    }else{
      for(let i = 0;i < ids.value.length;i++){
        let ind = _.findIndex(lids,(x)=>x === ids.value[i]);
        if(ind !== -1){
          _.pullAt(lids,ind);
        }
      }
    }
    lastIds.value = lids;
  }else{
    lastIds.value.push(...ids.value);
  }
  let resultIds = [];
  for(let i = 0;i < lastIds.value.length;i++){
    resultIds.push(lastIds.value[i]);
    let obj = _.find(tileNodes.value,function (v) {
      return v.id === lastIds.value[i];
    })
    if(obj.pid > 0){
      resultIds.push(obj.pid);
    }
  }
  roleForm.value.roles = _.uniq(resultIds);
}

function SearchByAccount(){
  roleList();
}

function changePage(page,cursor){
  page.value = page;
  cursor.value = cursor;
  Storage.SetItem("page",page);

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
  Storage.SetItem("roleId",roleApi.GetId("Setting.Role.Add"));
  let next = e.currentTarget.nextElementSibling;
  try {
    let res = await roleApi.Add(roleForm.value);
    next.style.display = "none";
    addRoleDetail.value.hide();
    await roleList();
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    next.style.display = "block";
    next.innerHTML = e;
    toastRef.value.show(e.message);
  }

}

async function editRole(){

  let res = await roleApi.Edit(roleId.value,roleForm.value);
  addRoleDetail.value.hide();
  toastRef.value.show("success");
  await roleList();
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
    toastRef.value.show("success");
    await roleList()
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
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