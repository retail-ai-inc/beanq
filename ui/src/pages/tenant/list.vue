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

            <a class="btn" href="javascript:;" @click="deleteTenantConfirm(item)"
               style="padding:.245rem .45rem;"
               data-bs-toggle="tooltip"
               data-bs-placement="top"
               data-bs-title="Delete"
               ref="deleteRef">
              <div class="icon-button" style="width: 1.125rem;height: 1.425rem;">
                <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="red" class="bi bi-trash" viewBox="0 0 16 16">
                  <path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm2.5 0a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5zm3 .5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6z"/>
                  <path fill-rule="evenodd" d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1v1zM4.118 4 4 4.059V13a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4.059L11.882 4H4.118zM2.5 3V2h11v1h-11z"/>
                </svg>
              </div>
            </a>

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
    <Action
        :label="deleteTenantLabel"
        :id="showTenantDeleteModal"
        :data-id="dataTenantId"
        :warning="$t('deleteRoleWarningHtml')"
        :info="info"
        @action="deleteTenantInfo">
      <template #title="{title}">
        Are you will delete this tenant?
      </template>
    </Action>
    <Btoast :id="toastId" ref="toastRef"></Btoast>
    <LoginModal :id="noticeId" ref="loginModal"/>
  </div>
</template>
<script setup>
import {ref,reactive,toRefs,onMounted} from "vue";
import i18n from "i18n";
import Btoast from "../components/btoast.vue";
import LoginModal from "../components/loginModal.vue";
import Action from "../components/action.vue";

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

let config = reactive({
  deleteTenantLabel:"deleteTenantLabel",
  showTenantDeleteModal:"showTenantDeleteModal",
  dataTenantId:"",
  id:"",
  info:"This operation will permanently delete the tenant. To avoid unintentional actions, please confirm by entering the tenant name:"
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

const deleteTenantConfirm= async (item)=>{

    config.deleteTenantLabel = "Delete Tenant";
    config.showTenantDeleteModal = "showTenantDeleteModal";
    config.dataTenantId = item.name;
    const ele = document.getElementById("showTenantDeleteModal");
    config.deleteTenantModal = new bootstrap.Modal(ele);
    config.deleteTenantModal.show(ele);
    config.dataTenantId = item.name;
    config.id = item.id;

}
const deleteTenantInfo = async ()=>{
  try {

    await tenantApi.Delete(config.id);
    await getTenants();
    toastRef.value.show(i18n.global.getLocaleMessage(Storage.GetItem("i18n") || "en")?.success);
    config.deleteTenantModal.hide();
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
const{deleteTenantLabel,showTenantDeleteModal,dataTenantId,info} = toRefs(config);

</script>
<style>
.list-group-item.active{
  background-color: #f8f9fa !important;
  border: none;
}
.list-group-item.active p{
  color: #0a584a !important;
}
</style>