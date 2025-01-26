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
                <button type="submit" class="btn btn-primary" @click="SearchByAccount">Search</button>
              </div>
              <div class="col-auto border-left" style="padding-left: 10px">
                <button type="button" class="btn btn-primary" @click="addUserModal">Add</button>
              </div>
            </div>
          </div>
        </div>
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
                  type="text"
                  class="form-control"
                  id="passwordInput"
                  v-model="userForm.password"
                  placeholder="The password length range is 5-36 chars"
                  @input="checkValid"
              />
              <div class="invalid-feedback">
                Please enter a length char of 5-36.
              </div>
            </div>
            <div class="mb-3">
              <label for="typeSelect" class="form-label">Type</label>
              <select class="form-select" aria-label="Type Select" id="typeSelect" v-model="userForm.type">
                <option selected>Open this select menu</option>
                <option value="normal">Normal</option>
                <option value="google">Google</option>
              </select>
            </div>
            <div class="mb-3">
              <label  class="form-label">Active</label>
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
            </div>
            <div class="mb-3">
              <label for="detailArea" class="form-label">Account Detail</label>
              <textarea class="form-control" id="detailArea" rows="3" v-model="userForm.detail"></textarea>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">{{l.closeButton}}</button>
            <button type="button" class="btn btn-primary" @click="addUser" v-if="accountReadOnly == false">{{l.addButton}}</button>
            <button type="button" class="btn btn-primary" @click="editUser" v-else>{{l.editButton}}</button>
            <div class="invalid-feedback">
            </div>
          </div>
        </div>
      </div>
    </div>
    <!--add user modal end-->

    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="userId" @action="deleteUser">
      <template #title="{title}">
        {{l.deleteModal.title}}
      </template>
    </Action>
    <Btoast :id="id" ref="toastRef">
    </Btoast>
  </div>
</template>
<script setup>
import { ref,inject,reactive,onMounted,toRefs,onUnmounted } from "vue";
import DeleteIcon from "../components/icons/delete_icon.vue";
import EditIcon from "../components/icons/edit_icon.vue";
import Pagination from "../components/pagination.vue";
import Action from "../components/action.vue";
import Btoast from "../components/btoast.vue";

const l = ref(inject("i18n"));

const [deleteLabel,delModal,showDeleteModal,account] = [ref("deleteLabel"),ref(null),ref("showDeleteModal"),ref("")];
const [id,toastRef] = [ref("userToast"),ref(null)];
const [users,accountReadOnly,addUserDetail] = [ref([]),ref(false),ref(null)];
const [page,pageSize,cursor,total] = [ref(1),ref(10),ref(0),ref(0)];
const [accountInput,userId] = [ref(""),ref("")];

let datas = reactive({
  userForm:{
    account:"",
    password:"",
    type:"normal",
    active:1,
    detail:""
  }
});

async function userList(){
  let res = await userApi.List(page.value,pageSize.value,accountInput.value);
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
    datas.userForm = {};
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

function SearchByAccount(){
  userList();
}

function changePage(page,cursor){
  page.value = page;
  cursor.value = cursor;
  sessionStorage.setItem("page",page)

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
    if(len < 5 || len > 36){
      next.innerHTML = "The password length range is 5-36 chars";
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
    let res = await userApi.Add(datas.userForm);
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
  let res = await userApi.Edit(datas.userForm);
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
  datas.userForm = item;
  accountReadOnly.value = true;
  const ele = document.getElementById("addUserDetail");

  addUserDetail.value = new bootstrap.Modal(ele);
  addUserDetail.value.show(ele);

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