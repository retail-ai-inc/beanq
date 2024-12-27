<template>
  <div class="container text-center" style="background: #f8f9fa">
    <div class="row align-items-start" style="height: 100vh;">
      <div class="col left-col">
        {{title}}
      </div>
      <div class="col right-col" >
        <div class="bq-box shadow p-3 mb-5 bg-body-tertiary rounded">
          <div style="width: 100%">
            <input class="form-control" type="text" placeholder="Username" name="userName" autocomplete="off" aria-label="default input example" v-model="user.username">
            <input class="form-control" type="password" placeholder="Password" name="password" autocomplete="off" aria-label="default input example" style="margin-top: 0.9375rem" v-model="user.password">
          </div>

          <button type="button" class="btn btn-primary" style="margin-top: 0.625rem" @click="onSubmit">Login</button>
          <div id="errorMsg" style="color: red;margin-top:0.625rem;">{{msg}}</div>

          <button type="button" class="btn btn-outline-secondary" @click="googleLogin" style="margin-top: 1rem;">
            <svg xmlns="http://www.w3.org/2000/svg" style="float: left;margin-top: 0.3rem;" width="16" height="16" fill="currentColor" class="bi bi-google" viewBox="0 0 16 16">
              <path d="M15.545 6.558a9.42 9.42 0 0 1 .139 1.626c0 2.434-.87 4.492-2.384 5.885h.002C11.978 15.292 10.158 16 8 16A8 8 0 1 1 8 0a7.689 7.689 0 0 1 5.352 2.082l-2.284 2.284A4.347 4.347 0 0 0 8 3.166c-2.087 0-3.86 1.408-4.492 3.304a4.792 4.792 0 0 0 0 3.063h.003c.635 1.893 2.405 3.301 4.492 3.301 1.078 0 2.004-.276 2.722-.764h-.003a3.702 3.702 0 0 0 1.599-2.431H8v-3.08h7.545z"/>
            </svg>
            <span>Sign In with Google</span>
          </button>

        </div>

`
      </div>
    </div>
  </div>
</template>
<script setup>
import { reactive,toRefs,onMounted,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';

const data = reactive({
  user:{"username":"","password":""},
  msg:"",
  title:config.title,
})
const useRe = useRouter();

onMounted(async ()=>{
  let token = useRe.currentRoute.value.query;
  if(JSON.stringify(token) !== "{}"){
    if (token.token != ""){
      await sessionStorage.setItem("token",token.token);
      useRe.push("/admin/home");
      return;
    }
  }
})

async function onSubmit(event){

  if (data.user.username == "" || data.user.password == ""){
    console.log("can not empty");
    return;
  }
  //,{headers:{"Content-Type":"multipart/form-data"}}
  try{
    let res = await loginApi.Login(data.user.username,data.user.password);
    sessionStorage.setItem("token",res.data.token);
    useRe.push("/admin/home");
  }catch(err){
    if (err.response.status === 401){
      data.msg = err.response.data.msg;
    }
  }
}

function googleLogin(){
  window.location.href="/googleLogin"
}

const {user,msg,title} = toRefs(data);
</script>
<style scoped>
.left-col{
  background: #7364dd;height: 100%;display: flex;justify-content: center;align-items: center;font-size: 1.5rem;font-weight: bold;color: #fff;
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
  flex-direction: column;
  justify-content: center;
  margin-left: 1.875rem;
}
</style>
