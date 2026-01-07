<template>
  <div class="tenant">
    <div class="row">
      <div class="col-3 mt-4">
        <h5 class="card-title d-flex flex-row justify-content-between">List of Tenants<button type="button" class="btn btn-primary btn-sm" @click="addTenant">Add Tenant</button></h5>
        <ul class="list-group list-group-flush">
          <li class="list-group-item" v-for="(item,key) in tenants" :key="key">
            <a @click="chooseTenant(item.id)" class="link-primary" style="cursor: pointer">{{item.name}}</a>
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
  </div>
</template>
<script setup>
import {ref,reactive,toRefs,onMounted,onUnmounted} from "vue";

let tenant = reactive({
  id:"1",
  name:"Trial",
  mongo:{
    host:"127.0.0.1",
    gcpHost:"sdfsfsfds",
    port:27017,
    name:"tenant-xxxxx-trial",
    userName:"aaa",
    userPwd:"bbb"
  },
  redis:{
    host:"127.0.0.1",
    gcpHost: "sdfsfsfds",
    port:6379,
    pwd:"aaaa"
  }
});
const currentUuid = ref("");
const tenants = ref([]);
const tenantModal = {add:ref(null),tenantName:ref(""),currentId:ref("")};


onMounted(async ()=>{

  await getTenants();

})

async function getTenants(){

  let res = await tenantApi.List(0,10,"","")
  const {rows,total} = res;
  tenants.value = rows;

}

const chooseTenant = (id)=>{
  currentUuid.value = id;
  console.log(currentUuid);
}

const updateTenantConfig=async ()=>{

  console.log(tenant);

  // try {
  //   let res = await request.put("",tenant);
  //   console.log(res);
  // }catch (e) {
  //   console.log(e);
  // }
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
  Object.assign(tenant,{
    id:"",
    name:"",
    mongo:{
      host:"",
      gcpHost:"",
      port:"",
      name:"",
      userName:"",
      userPwd:""
    },
    redis:{
      host:"",
      gcpHost: "",
      port:"",
      pwd:""
    }
  });
}

const {id,name,mongo,redis} = toRefs(tenant);

</script>