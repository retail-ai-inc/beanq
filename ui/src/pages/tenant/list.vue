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
    </div>
  </div>
</template>
<script setup>
import {ref,reactive,toRefs,onMounted,onUnmounted} from "vue";

const tenant = reactive({
  id:"1",
  name:"Trial",
  "mongo":{
    host:"127.0.0.1",
    gcpHost:"sdfsfsfds",
    port:27017,
    name:"tenant-xxxxx-trial",
    userName:"aaa",
    userPwd:"bbb"
  },
  "redis":{
    host:"127.0.0.1",
    gcpHost: "sdfsfsfds",
    port:6379,
    pwd:"aaaa"
  }
});
const currentUuid = ref("");
const tenants = ref([]);


onMounted(async ()=>{
  let arr = [{id:"1",name:"Trial"},{id:"2",name:"SiYo"}];
  tenants.value = arr;
  // try {
  //   let res = await request.get("tenant/config");
  //   tenant.mongo = res.mongo;
  //   tenant.redis = res.redis;
  // }catch (e) {
  //   console.log(e);
  // }
})

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
}

const {id,name,mongo,redis} = toRefs(tenant);

</script>