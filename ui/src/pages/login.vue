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

          <div id="recaptcha-v2" style="margin: 1rem 0;"></div>

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
  projectId:"",
  siteKeyV3:"",
  siteKeyV2:"",
  apiKey:"",
});
const googlereCAPTCHA = async(data)=>{

  Object.assign(googleReCaptcha.value, data);
  let active = data?.active;
  if(!active){
    return;
  }
  let siteKeyV3 = data?.siteKeyV3;
  if(siteKeyV3 === undefined){
    return;
  }
  await LoadScript(`https://www.google.com/recaptcha/enterprise.js?render=${siteKeyV3}`);

};

const debouncedHandleKeydown = Base.Debounce(handleKeyDown, 400);

onMounted(async ()=>{
  let {token=""} = useRe.currentRoute.value.query;

  if (token !== ""){
    await Storage.SetItem("token",token);
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

// google assessments
async function getGoogleRecaptchaAssessments(token,siteKey,projectId,apiKey){

  const body = {
    "event": {
      "token": token,
      "expectedAction": "USER_ACTION",
      "siteKey": `${siteKey}`,
    }
  };

  const url = `https://recaptchaenterprise.googleapis.com/v1/projects/${projectId}/assessments?key=${apiKey}`;
  return await axios.post(url, body,{headers:{"Content-Type":"application/json"}});

}

 async function showV2Captcha(v2SiteKey,projectId,apiKey){

    // clear container
    document.getElementById('recaptcha-v2').innerHTML = '';

    // render v2 reCAPTCHA
   grecaptcha.enterprise.ready(async()=>{
     grecaptcha.enterprise.render('recaptcha-v2', {
       'sitekey': v2SiteKey,
       'callback': async function(token) {

         const {status,data} = await getGoogleRecaptchaAssessments(token,v2SiteKey,projectId,apiKey);

         if(status !== 200){
           return false;
         }
         await login(formData.user.username,formData.user.password,expiredTimeBool.value);
         return true;
       }
     });
   })

}

async function onSubmit(event){

  event.preventDefault();

  disabled.value = true;
  let {username,password} = formData.user;

  if (username === "" || password === ""){
    msg.value = "Username or Password are required";
    disabled.value = false;
    return;
  }

  let {projectId,siteKeyV3,siteKeyV2,apiKey,active} = googleReCaptcha.value;

  if( !active || projectId === "" || siteKeyV3 === "" || siteKeyV2 === "" || apiKey === ""){
    await login(username,password,expiredTimeBool.value);
    return;
  }

  grecaptcha.enterprise.ready(async () => {
    try{
      const token = await grecaptcha.enterprise.execute(`${siteKeyV3}`, {action: 'LOGIN'});

      if(token === ""){
        return;
      }

      const {status,data} = await getGoogleRecaptchaAssessments(token,siteKeyV3,projectId,apiKey);
      if(status !== 200){
        return;
      }

      // check v3 verification result
      if(data.tokenProperties?.valid && (data.riskAnalysis?.score || 0) >= 0.5){
        // v3 verification passed, show v2 captcha
         await showV2Captcha(siteKeyV2,projectId,apiKey);
      } else {
        msg.value = "Verification failed, please try again";
        disabled.value = false;
      }
    }catch (e) {
      msg.value = "Verification error, please try again";
      disabled.value = false;
    }
  });

}

async function login(username,password,expiredTimeBool){

  try{
    let res = await loginApi.Login(username,password,expiredTimeBool);
    Storage.SetItem("token",res.token);
    Storage.SetItem("roles",res.roles);
    Storage.SetItem("nodeId",res.nodeId);

    let nodesRes = await dashboardApi.Nodes();
    Storage.SetItem("nodes",nodesRes);

    setTimeout(()=>{
      useRe.push("/admin/home");
    },1500)
  }catch(err){
    msg.value = err.response.data.msg;
    disabled.value = false;
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
