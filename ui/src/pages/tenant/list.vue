<template>
  <div class="tenant">
    <div class="row">
      <div class="col-3 mt-4">
        <h5 class="card-title d-flex flex-row justify-content-between">List of Tenants<button type="button" class="btn btn-primary btn-sm" @click="addTenant">Add Tenant</button></h5>
        <ul class="list-group list-group-flush">
          <li class="list-group-item d-flex flex-row justify-content-between align-items-center"
              :class="{active: currentUuid === item.id}" v-for="(item,key) in tenants" :key="key" >
            <p @click="chooseTenant(item.id)"
               :class="currentUuid === item.id ? 'text-white' : 'text-primary'" style="cursor: pointer;margin:0">{{item.name}}</p>
            <p class="text-danger" style="margin: 0;cursor:pointer" @click="deleteTenant(item.id)">{{$t("delete")}}</p>
          </li>
        </ul>
      </div>
      <div class="col mt-4">
        <div class="row">
          <div class="col">
            <div class="h5 mb-4 pb-2 border-bottom border-success-subtle">
              Basic
            </div>
            <div class="row mb-4">
              <div class="col-3 mb-3">
                <label for="tenant-name" class="form-label">Tenant Name</label>
                <input class="form-control" id="tenant-name" placeholder="Tenant Name" v-model="name" />
              </div>
            </div>

            <div class="h5 mb-4 pb-2 border-bottom border-success-subtle">
              Mongo
            </div>

            <div class="row mb-4">
              <div class="col-3 mb-3">
                  <label for="mongo-host" class="form-label">Host</label>
                  <input class="form-control" id="mongo-host" placeholder="host" v-model="mongo.host" />
              </div>
              <div class="col-3 mb-3">
                  <label for="mongo-gcp-host" class="form-label">GCP host</label>
                  <input class="form-control" id="mongo-gcp-host" placeholder="GCP host" v-model="mongo.gcpHost" />
              </div>
              <div class="col-3 mb-3">
                  <label for="mongo-port" class="form-label">Port</label>
                  <input class="form-control" id="mongo-port" placeholder="27017" v-model="mongo.port" />
              </div>
              <div class="col-3 mb-3">
                  <label for="mongo-name" class="form-label">DB name</label>
                  <input class="form-control" id="mongo-name" placeholder="tenant-xxxxx-trial" v-model="mongo.name" />
              </div>
              <div class="col-3 mb-3">
                  <label for="mongo-user-name" class="form-label">DB username</label>
                  <input class="form-control" id="mongo-user-name" placeholder="Username" v-model="mongo.userName" />
              </div>
              <div class="col-3 mb-3">
                  <label for="mongo-user-pwd" class="form-label">DB password</label>
                  <input class="form-control" id="mongo-user-pwd" placeholder="Password" v-model="mongo.userPwd" />
              </div>
            </div>
            <div class="h5 mb-4 pb-2 border-bottom border-success-subtle">
              Redis
            </div>
            <div class="row mb-4">
              <div class="col mb-3">
                  <label for="redis-host" class="form-label">Host</label>
                  <input class="form-control" id="redis-host" placeholder="127.0.0.1" v-model="redis.host" />
              </div>
              <div class="col mb-3">
                  <label for="redis-gcp-host" class="form-label">GCP host</label>
                  <input class="form-control" id="redis-gcp-host" placeholder="GCP host" v-model="redis.gcpHost" />
              </div>
              <div class="col mb-3">
                  <label for="redis-port" class="form-label">Port</label>
                  <input class="form-control" id="redis-port" placeholder="6379" v-model="redis.port" />
              </div>
              <div class="col mb-3">
                  <label for="redis-pwd" class="form-label">Password</label>
                  <input class="form-control" id="redis-pwd" placeholder="password" v-model="redis.pwd" />
              </div>
            </div>
            <button type="button" class="btn btn-primary" @click="updateTenantConfig">Update</button>

          </div>
          </div>
      </div>

      <!--add tenant Modal begin-->
      <div class="modal fade"
           id="staticBackdrop"
           data-bs-backdrop="static"
           data-bs-keyboard="false"
           tabindex="-1"
           aria-labelledby="staticBackdropLabel"
           aria-hidden="true">
        <div class="modal-dialog modal-dialog-centered">
          <div class="modal-content">
            <div class="modal-header">
              <h1 class="modal-title fs-5" id="staticBackdropLabel">Tenant Name</h1>
              <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
              <input class="form-control" id="tenant-modal-name" placeholder="Tenant Name" v-model="tenantModal.tenantName.value" />
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
              <button type="button" class="btn btn-primary" @click="doAddTenant">Submit</button>
            </div>
          </div>
        </div>
      </div>
      <!--add tenant Modal end-->


    </div>
    <Btoast :id="toastId" ref="toastRef"></Btoast>
    <LoginModal :id="noticeId" ref="loginModal"/>
  </div>
</template>
<script setup>
import {ref,reactive,toRefs,onMounted} from "vue";
import i18n from "i18n";
import Btoast from "../components/btoast.vue";
import LoginModal from "../components/loginModal.vue";

let tenant = reactive({
  id:"",
  name:"",
  mongo:{
    host:"",
    gcpHost:"",
    port:27017,
    name:"",
    userName:"",
    userPwd:""
  },
  redis:{
    host:"",
    gcpHost:"",
    port:6379,
    pwd:""
  }
});
const currentUuid = ref("");
const tenants = ref([]);
const tenantModal = {add:ref(null),tenantName:ref(""),currentId:ref("")};
const [noticeId,loginModal] = [ref("configBackdrop"),ref("loginModal")];
const [toastId,toastRef] = [ref("toast-" + Math.random().toString(36)),ref("toastRef")]

onMounted(async ()=>{

  await getTenants();

})

async function getTenants(){

  try{
    let res = await tenantApi.List(0,10,"","")
    const {rows,total} = res;
    if(rows.length > 0){
      tenants.value = rows;
      Object.assign(tenant,rows[0]);
      currentUuid.value = rows[0].id;
    }
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
}

const chooseTenant = async (id)=>{

  currentUuid.value = id;
  try{
    let res = await tenantApi.Get(id);
    Object.assign(tenant,res);
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }

}

const updateTenantConfig=async ()=>{

  try {
    await tenantApi.Update(currentUuid.value,tenant);
    toastRef.value.show(i18n.global.getLocaleMessage(Storage.GetItem("i18n") || "en")?.success);
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
}

const addTenant = async ()=>{

  tenantModal.add.value = new bootstrap.Modal(document.getElementById("staticBackdrop"));
  tenantModal.add.value.show();

}

const doAddTenant = async ()=>{

   let res = await tenantApi.Add({name:tenantModal.tenantName.value});
   tenantModal.currentId.value = res.id;
   await getTenants();
   tenantModal.add.value.hide();
   tenant = {id:res.id,name:tenantModal.tenantName.value,mongo:{},redis:{}};

}

const deleteTenant= async (id)=>{
  try {
    await tenantApi.Delete(id);
    await getTenants();
    toastRef.value.show(i18n.global.getLocaleMessage(Storage.GetItem("i18n") || "en")?.success);
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
}
const {id,name,mongo,redis} = toRefs(tenant);

</script>