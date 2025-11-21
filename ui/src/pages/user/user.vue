<template>
  <div class="user">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of Admin Users</h5>
      </div>
    </div>
    <div class="row">
      <div class="col-12">
        <div class="row">
          <div class="col">
            <div class="form-row mb-3">
              <div class="col">
                <input type="text" class="form-control" id="formId" name="formId" v-model="accountInput" placeholder="Search by account">
              </div>
              <div class="col-auto" style="margin:0 .75rem;">
                <button type="submit" class="btn btn-primary" @click="SearchByAccount">{{$t('search')}}</button>
              </div>
              <div class="col-auto border-left" style="padding-left: 10px">
                <button type="button" class="btn btn-primary" @click="addUserModal">{{$t('add')}}</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <Spinner v-if="loading"/>
    <div v-else>
      <NoMessage v-if="users.length <= 0">
        <template #content="{content}">
          There is no admin, please create one.
        </template>
      </NoMessage>
      <div v-else>
        <Pagination :page="page" :total="total" :cursor="cursor" @changePage="changePage"/>
        <table class="table table-striped table-hover" style="table-layout: auto;">
          <thead>
          <tr>
            <th scope="col" class="w-table-number">#</th>
            <th scope="col" class="text-center">Id</th>
            <th scope="col" class="text-center">Account</th>
            <th scope="col" class="text-center">Active</th>
            <th scope="col" class="text-center">Type</th>
            <th scope="col" class="col-4 text-center">Detail</th>
            <th scope="col" class="col-2 text-center">Action</th>
          </tr>
          </thead>
          <tbody>
          <tr v-for="(item, key) in users" :key="key" style="height: 3rem;line-height:3rem">
            <td>{{key+1}}</td>
            <td class="text-center">{{item._id}}</td>
            <td class="text-center">{{item.account}}</td>
            <td class="text-center">
              <span :class="item.active == 1 ? 'green' : 'red'">{{item.active == "1" ? "active" :"locked"}}</span>
            </td>
            <td class="text-center">{{item.type}}</td>
            <td class="text-center">
          <span class="d-inline-block text-truncate" style="max-width: 5rem;">
            {{item.detail}}
          </span>
            </td>
            <td class="text-end text-nowrap">
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
    <div class="modal fade" id="addUserDetail" data-bs-keyboard="false" tabindex="-1" aria-labelledby="addUserDetailLabel">
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h1 class="modal-title fs-5" id="addUserDetailLabel">
              {{accountReadOnly == true ? "Edit User" : "Add User"}}
              </h1>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
          </div>
          <div class="modal-body">
            <div class="mb-3">
              <label for="accountInput" class="form-label">Account ID
              </label>
              <input
                  type="text"
                  class="form-control"
                  id="accountInput"
                  @blur="checkValid"
                  v-model="userForm.account"
                  :readonly="accountReadOnly == true ? 'readonly': false"
                  :disabled="accountReadOnly === true ? 'disabled': false"
                  placeholder="Account ID should be an email"
              />
              <div class="invalid-feedback">
                Please input an account.
              </div>
            </div>
            <div class="mb-3">
              <label for="passwordInput" class="form-label">Password</label>
              <input
                  name="passwordInput"
                  type="text"
                  class="form-control"
                  id="passwordInput"
                  v-model="userForm.password"
                  placeholder="The password length range is 5-36 chars"
                  @input="checkValid"
              />
              <div class="invalid-feedback">
                password must be 5-36 characters. must requirea symbol  must required atleast one uppercase letter.
              </div>
            </div>
            <div class="mb-3">
              <label for="typeSelect" class="form-label">Type</label>
              <select class="form-select" aria-label="Type Select" id="typeSelect" name="typeSelect" v-model="userForm.type">
                <option selected>Open this select menu</option>
                <option value="normal">Normal</option>
                <option value="google">Google</option>
              </select>
            </div>
            <div class="mb-3">
              <label for="roleSelect" class="form-label">Role</label>
              <select class="form-select" id="roleSelect" name="roleSelect" v-model="userForm.roleId" v-if="roles.length > 0">
                <option v-for="(item,key) in roles" :value="item._id" :key="key" :selected="userForm.roleId === item._id">{{item.name}}</option>
              </select>
              <div v-else>
                <div>Please add a role by clicking <router-link to="/admin/role" class="btn text-primary">here</router-link> first</div>
              </div>

            </div>
            <div class="mb-3">
              <label class="form-label">
                Active
                <div class="form-check">
                  <input class="form-check-input" type="radio" name="flexRadioDefault" id="flexRadioDefault1" value="1" v-model="userForm.active" :checked="userForm.active == 1">
                  <label class="form-check-label" for="flexRadioDefault1">
                    Yes
                  </label>
                </div>
                <div class="form-check">
                  <input class="form-check-input" type="radio" name="flexRadioDefault" id="flexRadioDefault2" value="2" v-model="userForm.active" :checked="userForm.active == 2">
                  <label class="form-check-label" for="flexRadioDefault2">
                    No
                  </label>
                </div>
              </label>

            </div>
            <div class="mb-3">
              <label for="detailArea" class="form-label">Account Detail</label>
              <textarea class="form-control" id="detailArea" name="detailArea" rows="3" v-model="userForm.detail"></textarea>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">{{$t('close')}}</button>
            <button type="button" class="btn btn-primary" @click="addUser" v-if="accountReadOnly == false">{{$t('add')}}</button>
            <button type="button" class="btn btn-primary" @click="editUser" v-else>{{$t('edit')}}</button>
            <div class="invalid-feedback">
            </div>
          </div>
        </div>
      </div>
    </div>
    <!--add user modal end-->

    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="userId" :warning="$t('deleteRoleWarningHtml')" :info="$t('deleteRoleInfoHtml')" @action="deleteUser">
      <template #title="{title}">
        Are you sure you want to delete the user?
      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
    <LoginModal :id="loginId" ref="loginModal"/>

  </div>
</template>
<script setup>
import { ref,reactive,computed,onMounted,toRefs,onUnmounted } from "vue";
import DeleteIcon from "../components/icons/delete_icon.vue";
import EditIcon from "../components/icons/edit_icon.vue";
import Pagination from "../components/pagination.vue";
import Action from "../components/action.vue";
import Btoast from "../components/btoast.vue";
import LoginModal from "../components/loginModal.vue";
import NoMessage from "../components/noMessage.vue";
import Spinner from "../components/spinner.vue";

const nav = computed(()=>{
  return Nav;
})

const [deleteLabel,delModal,showDeleteModal,account] = [ref("deleteLabel"),ref(null),ref("showDeleteModal"),ref("")];
const [id,toastRef] = [ref("userToast"),ref(null)];
const [users,accountReadOnly,addUserDetail] = [ref([]),ref(false),ref(null)];
const [page,pageSize,cursor,total] = [ref(1),ref(10),ref(0),ref(0)];
const [accountInput,userId] = [ref(""),ref("")];
const [loading] = [ref(false)];

const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];


let datas = reactive({
  userForm:{
    account:"",
    password:"",
    type:"normal",
    active:1,
    roleId:"",
    detail:""
  }
});
const roles = ref([]);

async function roleList(){
  try {
    let res = await roleApi.List(0,100);

    if(res.data !== null){
      roles.value = res.data;
    }

  }catch (e) {
    console.log(e.status)
    toastRef.value.show(e);
  }

}

async function userList(){
  loading.value = true;
  try {
    let res = await userApi.List(page.value,pageSize.value,accountInput.value);
    users.value = res.data ?? [];
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
   userList();
  const ele = document.getElementById("addUserDetail");
  ele.addEventListener('hidden.bs.modal', () => {
    datas.userForm = {};
    accountReadOnly.value = false;
  });

})

onUnmounted(()=>{
  document.querySelectorAll(".modal-backdrop").forEach(el => el.remove());
})

function SearchByAccount(){
  userList();
}

function changePage(page,cursor){
  page.value = page;
  cursor.value = cursor;
  Storage.SetItem("page",page)

  userList();
}

function checkValid(e){

  let next = e.currentTarget.nextElementSibling;
  next.style.display = "none";
  //check account
  if(e.currentTarget.id === "accountInput"){
    next.innerHTML = "Please input an Email account";
    if(datas.userForm.account === "" || Base.CheckEmail(datas.userForm.account) === false){
      next.style.display = "block";
    }
  }
  //check password
  if(e.currentTarget.id === 'passwordInput'){
    let len = datas.userForm.password.length;
    if(len <= 0){
      next.innerHTML = "";
      next.style.display = "none";
      return;
    }
    const regex = /^(?=.*[A-Z])(?=.*[^A-Za-z0-9\s]).{5,36}$/;
    if(!regex.test(datas.userForm.password)){
      next.style.display = "block";
      next.style.color = "#dc3545";
      next.innerHTML = "password must be 5-36 characters. must require symbol . must required at least one uppercase letter.";
    }
  }
}


function addUserModal(){
  const ele = document.getElementById("addUserDetail");
  addUserDetail.value = new bootstrap.Modal(ele);
  addUserDetail.value.show(ele);
  roleList();
}

async function addUser(e){

  try {
    await userApi.Add(datas.userForm);

    addUserDetail.value.hide();
    toastRef.value.show("success");
    setTimeout(async ()=>{
      await userList();
    },3000);

  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e.message);
  }
  
}

async function editUser(){
  try {
    let res = await userApi.Edit(datas.userForm);
    addUserDetail.value.hide();
    toastRef.value.show("success");
    await userList();
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e.error);
  }
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
    console.log(res);
    toastRef.value.show("success");
    await userList();
  }catch (e) {
    if(e.status === 401){
      loginModal.value.error(new Error(e));
      return
    }
    toastRef.value.show(e.message);
  }

}

function editUserModal(item){
  datas.userForm = item;
  accountReadOnly.value = true;
  const ele = document.getElementById("addUserDetail");

  addUserDetail.value = new bootstrap.Modal(ele);
  addUserDetail.value.show(ele);

  roleList();
}

const {userForm} = toRefs(datas);
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