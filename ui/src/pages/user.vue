<template>
  <div class="user">
    <div class="d-flex justify-content-end">
      <button type="button" class="btn btn-primary" @click="addUserModal">Add</button>
    </div>
    <table class="table table-striped table-hover">
      <thead>
      <tr>
        <th scope="col">#</th>
        <th scope="col">Account</th>
        <th scope="col">Active</th>
        <th scope="col">Type</th>
        <th scope="col">Detail</th>
        <th scope="col">Action</th>
      </tr>
      </thead>
      <tbody>
      <tr v-for="(item, key) in users" :key="key" style="height: 3rem;line-height:3rem">
        <th scope="row">{{parseInt(key)+1}}</th>
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
        <td>
            <a href="javascript:;" @click="editUserModal(item)">Edit</a> <div class="vr"></div> <a href="javascript:;" @click="deleteUser(item)">Delete</a>
        </td>
      </tr>
      </tbody>

    </table>

    <!--add user modal-->
    <div class="modal fade" id="addUserDetail" data-bs-keyboard="false" tabindex="-1" aria-labelledby="addUserDetailLabel" aria-hidden="true">
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
              <label for="accountInput" class="form-label">Account</label>
              <input
                  type="text"
                  class="form-control"
                  id="accountInput"
                  @blur="checkValid"
                  v-model="userForm.account"
                  :readonly="accountReadOnly == true ? 'readonly': false"/>
              <div class="invalid-feedback">
                Please input an account.
              </div>
            </div>
            <div class="mb-3">
              <label for="passwordInput" class="form-label">Password</label>
              <input type="text" class="form-control" id="passwordInput" v-model="userForm.password"/>
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
              <label for="detailArea" class="form-label">Detail</label>
              <textarea class="form-control" id="detailArea" rows="3" v-model="userForm.detail"></textarea>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
            <button type="button" class="btn btn-primary" @click="addUser" v-if="accountReadOnly == false">Add</button>
            <button type="button" class="btn btn-primary" @click="editUser" v-else>Edit</button>
            <div class="invalid-feedback">
            </div>
          </div>
        </div>
      </div>
    </div>
    <!--add user modal end-->

  </div>
</template>
<script setup>
import { reactive,onMounted,toRefs,onUnmounted } from "vue";

let data = reactive({
  users:[],
  userForm:{
    account:"",
    password:"",
    type:"normal",
    active:1,
    detail:""
  },
  accountReadOnly:false,
  addUserDetail:null,
})

onMounted(async ()=>{
  let res = await userApi.List();
  data.users = res.data;

  const ele = document.getElementById("addUserDetail");
  ele.addEventListener('hidden.bs.modal', () => {
    data.userForm = {};
    data.accountReadOnly = false;
  });

})

onUnmounted(()=>{
    const ele = document.getElementById('addUserDetail');
    if (ele) {
      ele.removeEventListener('hidden.bs.modal', () => {

      });
    }
})

function checkValid(e){

  //check account
  if(e.currentTarget.id == "accountInput"){
    let next = e.currentTarget.nextElementSibling;
    next.style.display = "none";
    next.innerHTML = "Please input an account";
    if(data.userForm.account == ""){
      next.style.display = "block";
    }
  }
}

function addUserModal(){
  const ele = document.getElementById("addUserDetail");
  data.addUserDetail = new bootstrap.Modal(ele);
  data.addUserDetail.show(ele);
}

async function addUser(e){

  let next = e.currentTarget.nextElementSibling;
  let res = await userApi.Add(data.userForm);
  if(res.code != "0000"){
    next.style.display = "block";
    next.innerHTML = res.msg;
    return
  }
  next.style.display = "none";
  data.addUserDetail.hide();
  let users = await userApi.List();
  data.users = users.data;

}

async function editUser(){
  console.info(data.userForm);
  let res = await userApi.Edit(data.userForm);
  data.addUserDetail.hide();
  let users = await userApi.List();
  data.users = users.data;
  return
}

async function deleteUser(item){

  let res = await userApi.Delete(item.account);
  if(res.code == "0000"){
    let res = await userApi.List();
    data.users = res.data;
    return
  }
}

function editUserModal(item){
  data.userForm = item;
  data.accountReadOnly = true;
  const ele = document.getElementById("addUserDetail");

  data.addUserDetail = new bootstrap.Modal(ele);
  data.addUserDetail.show(ele);

  console.log(data.userForm);
}

const {users,userForm,accountReadOnly} = toRefs(data);
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