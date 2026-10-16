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

          <button class="btn btn-primary" type="button" :disabled="disabled" @click="onSubmit">

            <span v-if="disabled" class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
            <span v-if="disabled">Loading...</span>
            <span v-else>Login</span>
          </button>

          <div id="errorMsg" style="color: red;margin-top:0.625rem;text-align: left">{{msg}}</div>

          <button type="button" class="btn btn-outline-secondary" @click="googleLogin" style="margin-top: 1rem;" v-if="showGoogleLogin">
            <svg xmlns="http://www.w3.org/2000/svg" style="float: left;margin-top: 0.3rem;" width="16" height="16" fill="currentColor" class="bi bi-google" viewBox="0 0 16 16">
              <path d="M15.545 6.558a9.42 9.42 0 0 1 .139 1.626c0 2.434-.87 4.492-2.384 5.885h.002C11.978 15.292 10.158 16 8 16A8 8 0 1 1 8 0a7.689 7.689 0 0 1 5.352 2.082l-2.284 2.284A4.347 4.347 0 0 0 8 3.166c-2.087 0-3.86 1.408-4.492 3.304a4.792 4.792 0 0 0 0 3.063h.003c.635 1.893 2.405 3.301 4.492 3.301 1.078 0 2.004-.276 2.722-.764h-.003a3.702 3.702 0 0 0 1.599-2.431H8v-3.08h7.545z"/>
            </svg>
            <span>Sign In with Google</span>
          </button>

        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import { ref,reactive,toRefs,onMounted,onUnmounted } from "vue";
import { useRouter } from 'vueRouter';

const LOGIN_ACTION = "LOGIN";

const [formData,useRe] = [
    reactive({
      user:{"username":"","password":""},
      title:config.title,
    }),
  useRouter()
];
const expiredTimeBool = ref(false);

const [disabled,msg] = [ref(false),ref("")];

function handleKeyDown(event){
  if(event.key === "Enter"){
    onSubmit(event)
  }
}

const getConfig = async ()=>{
  return await loginApi.AllowGoogle();
}

const showGoogleLogin = ref(false);
const allowGoogle = async (data)=>{
    showGoogleLogin.value = (!!data?.clientId) && (!!data?.clientSecret);
}

const googleReCaptcha = ref({
  active:false,
  siteKey:""
});
const googlereCAPTCHA = async(data)=>{
  const active = !!data?.active;
  const siteKey = data?.siteKey || "";
  googleReCaptcha.value = {active, siteKey};
  if(!active || siteKey === ""){
    return;
  }
  await LoadScript(`https://www.google.com/recaptcha/enterprise.js?render=${encodeURIComponent(siteKey)}`);
};

const debouncedHandleKeydown = Base.Debounce(handleKeyDown, 400);

onMounted(async ()=>{
	let {authenticated=""} = useRe.currentRoute.value.query;

	if (authenticated === "1"){
	await Storage.SetItem("authenticated","1");
    useRe.push("/admin/home");
    return;
  }

  let {google,googleReCAPTCHA} = await getConfig();

  await allowGoogle(google);
  await googlereCAPTCHA(googleReCAPTCHA);

  window.addEventListener("keydown",debouncedHandleKeydown);

})

onUnmounted(()=>{
  window.removeEventListener("keydown",debouncedHandleKeydown)
})

function executeFraudDefense(siteKey){
  return new Promise((resolve, reject) => {
    if (!window.grecaptcha?.enterprise) {
      reject(new Error("Google Cloud Fraud Defense is not loaded"));
      return;
    }
    grecaptcha.enterprise.ready(async () => {
      try {
        const token = await grecaptcha.enterprise.execute(siteKey, {action: LOGIN_ACTION});
        resolve(token);
      } catch (e) {
        reject(e);
      }
    });
  });
}

async function onSubmit(event){
  event.preventDefault();

  disabled.value = true;
  msg.value = "";
  let {username,password} = formData.user;

  if (username === "" || password === ""){
    msg.value = "Username or Password are required";
    disabled.value = false;
    return;
  }

  let {siteKey,active} = googleReCaptcha.value;
  let recaptchaToken = "";

  if(active && siteKey !== ""){
    try{
      recaptchaToken = await executeFraudDefense(siteKey);
      if(!recaptchaToken){
        msg.value = "Verification failed, please try again";
        disabled.value = false;
        return;
      }
    }catch (e) {
      msg.value = "Verification error, please try again";
      disabled.value = false;
      return;
    }
  }

  await login(username,password,expiredTimeBool.value, recaptchaToken);
}

async function login(username,password,expiredTimeBool, recaptchaToken){
  try{
    let res = await loginApi.Login(username,password,expiredTimeBool, recaptchaToken);
	Storage.SetItem("authenticated","1");
    Storage.SetItem("roles",res.roles);
    Storage.SetItem("nodeId",res.nodeId);

    let nodesRes = await dashboardApi.Nodes();
    Storage.SetItem("nodes",nodesRes);

    setTimeout(()=>{
      useRe.push("/admin/home");
    },1500)
  }catch(err){
	msg.value = err.response?.data?.msg || err.message || "Login failed";
    disabled.value = false;
  }
}

function googleLogin(){
	window.location.href="/api/v1/auth/google"
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
