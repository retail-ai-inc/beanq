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
          <div class="form-check d-flex" style="margin:1rem 0">
            <input class="form-check-input" type="checkbox" v-model="expiredTimeBool" id="flexCheckDefault">
            <label class="form-check-label" for="flexCheckDefault" style="margin-left: .4rem">
              Free login within 30 days
            </label>
          </div>

          <button type="button" class="btn btn-primary" style="margin-top: 0.625rem" @click="onSubmit">Login</button>
          <div id="errorMsg" style="color: red;margin-top:0.625rem;text-align: left">{{msg}}</div>

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
import { ref,reactive,toRefs,onMounted,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';

const [formData,useRe] = [
    reactive({
      user:{"username":"","password":""},
      title:config.title,
    }),
  useRouter()
];
const expiredTimeBool = ref(false);

const msg = ref("");

function handleKeyDown(event){
  if(event.key === "Enter"){
    onSubmit(event)
  }
}
const debouncedHandleKeydown = Base.Debounce(handleKeyDown, 400);

onMounted(async ()=>{
  let token = useRe.currentRoute.value.query;
  if(JSON.stringify(token) !== "{}"){
    if (token.token != ""){
      await sessionStorage.setItem("token",token.token);
      //useRe.push("/admin/home");
      return;
    }
  }
  window.addEventListener("keydown",debouncedHandleKeydown)
})

onUnmounted(()=>{
  window.removeEventListener("keydown",debouncedHandleKeydown)
})


async function onSubmit(event){

  if (formData.user.username == "" || formData.user.password == ""){
    msg.value = "Username or Password are required";
    return;
  }

  //,{headers:{"Content-Type":"multipart/form-data"}}
  try{
    let res = await loginApi.Login(formData.user.username,formData.user.password,expiredTimeBool.value);
    sessionStorage.setItem("token",res.token);
    sessionStorage.setItem("roles",res.roles);
    sessionStorage.setItem("nodeId",res.nodeId);

    let nodesRes = await dashboardApi.Nodes();
    sessionStorage.setItem("nodes",nodesRes);

    useRe.push("/admin/home");
  }catch(err){
    msg.value = err.response.data.msg;
  }
}


function googleLogin(){
  window.location.href="/googleLogin"
}

const {user,title} = toRefs(formData);
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
