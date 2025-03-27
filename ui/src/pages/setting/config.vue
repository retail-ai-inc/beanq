<template>
  <div class="config container-fluid">
      <div class="row mb-4">
        <div class="col">
          <h5 class="">Configuration Information</h5>
        </div>
      </div>
      <div class="row">
        <div class="col">
          <span>Google Credential</span>
          <hr/>

          <div class="container">
            <div class="row g-3 align-items-center m-2">
              <div class="col-1 text-end">
                <label for="clientId" class="col-form-label">ClientID:</label>
              </div>
              <div class="col-6">
                <input type="text" id="clientId" class="form-control" v-model="form.google.clientId" aria-describedby="passwordHelpInline" required>
              </div>
              <div class="col-3">
          <span id="clientIdHelpInline" class="form-text">
            create <a :href="credentials" target="_blank">credentials</a>
          </span>
              </div>
            </div>

            <div class="row g-3 align-items-center m-2">
              <div class="col-1 text-end">
                <label for="clientSecret" class="col-form-label">ClientSecret:</label>
              </div>
              <div class="col-6">
                <input type="text" id="clientSecret" class="form-control" v-model="form.google.clientSecret" aria-describedby="passwordHelpInline" required>
              </div>
              <div class="col-3">
          <span id="clientSecretHelpInline" class="form-text">

          </span>
              </div>
            </div>

            <div class="row g-3 align-items-center m-2">
              <div class="col-1 text-end">
                <label for="callBackUrl" class="col-form-label">CallBackURL:</label>
              </div>
              <div class="col-6">

                <div class="input-group mb-3">
                  <button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">{{schemeVal}}</button>
                  <ul class="dropdown-menu">
                    <li><a class="dropdown-item" href="javascript:;" @click="scheme('Https')">Https</a></li>
                    <li><a class="dropdown-item" href="javascript:;" @click="scheme('Http')">Http</a></li>
                  </ul>
                  <input type="url" id="callBackUrl" class="form-control" v-model="form.google.callBackUrl" aria-label="Text input with dropdown button" required>
                </div>

              </div>
              <div class="col-3">
          <span id="callBackUrlHelpInline" class="form-text">

          </span>
              </div>
            </div>
          </div>

          <span>SMTP Information</span>
          <hr/>

          <div class="container">
          </div>

          <span>Alert Rule</span>
          <hr/>
          <div class="container">
            <div class="row g-3 align-items-center m-2">
              <div class="col-1 text-end">
                <label for="alertRule" class="col-form-label">When:</label>
              </div>
              <div class="col-6">
                an event is captured by Sentry and all of the following happens
              </div>
            </div>
            <div class="row g-3 align-items-center m-2">
              <div class="col-1 text-end">
                <label for="alertRule" class="col-form-label">IF:</label>
              </div>
              <div class="col-6">
                an event is captured by Sentry and all of the following happens
              </div>
            </div>
            <div class="row g-3 align-items-center m-2">
              <div class="col-1 text-end">
                <label for="alertRule" class="col-form-label">Then:</label>
              </div>
              <div class="col-6">
                an event is captured by Sentry and all of the following happens
              </div>
            </div>
          </div>


          <button type="button" class="btn btn-primary" @click="edit">{{$t('edit')}}</button>
        </div>
      </div>
    <Btoast :id="toastId" ref="toastRef"></Btoast>
    <LoginModal :id="noticeId" ref="loginModal"/>
  </div>
</template>
<script setup>
import {ref,watch,onMounted} from 'vue';
import Btoast from "../components/btoast.vue";
import LoginModal from "../components/loginModal.vue";
import i18n from "i18n";

const [noticeId,loginModal] = [ref("configBackdrop"),ref("loginModal")];
const [toastId,toastRef] = [ref("toast-" + Math.random().toString(36)),ref("toastRef")]
const credentials = ref("https://console.cloud.google.com/apis/credentials?pli=1&inv=1&invt=Abs9TA");
const schemeVal = ref("Https");
const form = ref({
  google:{
    clientId: "",
    clientSecret: "",
    callBackUrl: "",
  },
  smtp:{
    host: "",
    port: "",
    user: "",
    password: "",
  },
  sendGrid:{
    key:"",
    fromName:"",
    fromAddress:""
  }
});

onMounted(()=>{
  list();
})

watch(() => form.value.google.clientId, (n, o) => {
  let ele = document.getElementById("clientId");
  if(n !== ""){
    ele.style.cssText = "border-color: #ced4da;";
  }else {
    ele.style.cssText = "border-color: red;";
  }
})

watch(()=> form.value.google.clientSecret, (n, o) => {
  let ele = document.getElementById("clientSecret");
  if(n !== ""){
    ele.style.cssText = "border-color: #ced4da;";
  }else {
    ele.style.cssText = "border-color: red;";
  }
})

watch(()=> form.value.google.callBackUrl, (n, o) => {
  let ele = document.getElementById("callBackUrl");
  if(n !== ""){
    ele.style.cssText = "border-color: #ced4da;";
  }else {
    ele.style.cssText = "border-color: red;";
  }
})

const scheme = (val) => {
  schemeVal.value = val;
}

const list = async () => {
  try {
    let res = await configApi.getConfig();
    form.value.google = JSON.parse(res.google);
    form.value.smtp = JSON.parse(res.smtp);
    form.value.sendGrid = JSON.parse(res.sendGrid);

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

const edit = async () => {

  if (form.value.google.clientId === "") {
    document.getElementById("clientId").style.cssText = "border-color: red;";
    return;
  }
  if(form.value.google.clientSecret === ""){
    document.getElementById("clientSecret").style.cssText = "border-color: red;";
    return;
  }
  if(form.value.google.callBackUrl === ""){
    document.getElementById("callBackUrl").style.cssText = "border-color: red;";
    return;
  }

  let result = {
    google:{
      clientId: form.value.google.clientId,
      clientSecret: form.value.google.clientSecret,
      callBackUrl: schemeVal.value + "://" + form.value.google.callBackUrl,
    }
  }

  try {
    let res = await configApi.updateConfig(result);
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
</script>