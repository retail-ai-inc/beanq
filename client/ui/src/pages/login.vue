<template>
  <div class="container text-center" style="background: #f8f9fa">
    <div class="row align-items-start" style="height: 100vh;">
      <div class="col left-col">
        Welcome To Beanq Monitor UI
      </div>
      <div class="col right-col" >
        <div class="bq-box">
          <div style="width: 100%">
            <input class="form-control" type="text" placeholder="Username" aria-label="default input example" v-model="user.username">
            <input class="form-control" type="password" placeholder="Password" aria-label="default input example" style="margin-top: 15px" v-model="user.password">
          </div>

          <button type="button" class="btn btn-primary" style="margin-top: 10px" @click="onSubmit">Login</button>
          <div id="errorMsg" style="color: red;margin-top:10px;">{{msg}}</div>
        </div>


      </div>
    </div>
  </div>
</template>
<script setup>
import { reactive,toRefs,onMounted,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';

import request  from "request";

const data = reactive({
  user:{"username":"","password":""},
  msg:""
})
const useRe = useRouter();

function onSubmit(){
  if (data.user.username == "" || data.user.password == ""){
    console.log("can not empty");
    return;
  }
  request.post("/login", {username:data.user.username,password:data.user.password},{headers:{"Content-Type":"multipart/form-data"}} ).then(res=>{
    sessionStorage.setItem("token",res.data.token);
    useRe.push("/admin/home");
  }).catch(err=>{
    if (err.response.status == 401){
      data.msg = err.response.data.msg;
    }
  })
}
const {user,msg} = toRefs(data);
</script>
<style scoped>
.left-col{
  background: #7364dd;height: 100%;display: flex;justify-content: center;align-items: center;font-size: 24px;font-weight: bold;color: #fff;
}
.right-col{
  display: flex;
  flex-direction: column;
  justify-content: center;
  height: 100vh;
}
.bq-box{
  display: flex;width: 70%;
  background: #fff;
  padding: 25px;
  border:1px solid #ced4da;
  border-radius: 5px;
  box-shadow:4px 4px 5px -6px;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  margin-left: 30px;
}
</style>
