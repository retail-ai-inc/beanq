<template>
  <div class="config container-fluid">
      <div class="row mb-4">
        <div class="col">
          <h5 class="">Configuration Information</h5>
        </div>
      </div>
      <div class="row">
        <div class="col">

          <nav>
            <div class="nav nav-tabs" id="nav-tab" role="tablist">
              <button class="nav-link active" id="google-tab" data-bs-toggle="tab" data-bs-target="#google-pan" type="button" role="tab" aria-controls="google-pan" aria-selected="true">Google Credential</button>
              <button class="nav-link" id="smtp-tab" data-bs-toggle="tab" data-bs-target="#smtp-pan" type="button" role="tab" aria-controls="smtp-pan" aria-selected="false">SMTP</button>
              <button class="nav-link" id="send-grid-tab" data-bs-toggle="tab" data-bs-target="#send-grid-pane" type="button" role="tab" aria-controls="send-grid-pane" aria-selected="false">SendGrid</button>
              <button class="nav-link" id="slack-tab" data-bs-toggle="tab" data-bs-target="#slack-pane" type="button" role="tab" aria-controls="slack-pane" aria-selected="false">Slack</button>
              <button class="nav-link" id="alert-rule-tab" data-bs-toggle="tab" data-bs-target="#alert-rule-pane" type="button" role="tab" aria-controls="alert-rule-pane" aria-selected="false">Alert Rule</button>
            </div>
          </nav>


          <div class="tab-content" id="myTabContent">
            <!--google credential-->
            <Google  class="tab-pane fade show active"
                     id="google-pan"
                     role="tabpanel" aria-labelledby="google-tab" tabindex="0"
                     v-model="form.google"
            />

            <!--smtp-->
            <Smtp class="tab-pane fade"
                  id="smtp-pan"
                  role="tabpanel"
                  aria-labelledby="smtp-tab"
                  tabindex="0"
                  v-model="form.smtp"
            />

            <!--send grid-->
            <SendGrid class="tab-pane fade"
                      id="send-grid-pane"
                      role="tabpanel"
                      aria-labelledby="send-grid-tab"
                      tabindex="0"
                      v-model="form.grid"
            />
            <Slack class="tab-pane fade"
                      id="slack-pane"
                      role="tabpanel"
                      aria-labelledby="slack-tab"
                      tabindex="0"
                      v-model="form.slack"
            />
            <!--alert rule-->
            <AlertRule class="tab-pane fade"
                      id="alert-rule-pane"
                      role="tabpanel"
                      aria-labelledby="alert-rule-tab"
                      tabindex="0"
                      v-model="form.rule"
                      @onTestNotify="onTestNotify"
            />
            <button type="button" class="btn btn-primary" @click="edit" style="margin-top: 2rem;">{{$t('edit')}}</button>
          </div>
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
import Delete_icon from "./delete_icon.vue";
import Google from "./config/google.vue";
import Smtp from "./config/smtp.vue";
import SendGrid from "./config/sendGrid.vue";
import AlertRule from "./config/alertRule.vue";
import Slack from "./config/slack.vue";

const [noticeId,loginModal] = [ref("configBackdrop"),ref("loginModal")];
const [toastId,toastRef] = [ref("toast-" + Math.random().toString(36)),ref("toastRef")]

const form = ref({
  google:{
    clientId: "",
    clientSecret: "",
    callBackUrl: "",
    scheme:""
  },
  smtp:{
    host: "",
    port: "",
    user: "",
    password: "",
  },
  grid:{
    key:"",
    fromName:"",
    fromAddress:""
  },
  slack:{
    botAuthToken:""
  },
  rule:{
    when:[],
    if:[],
    then:[]
  }
});


const [triggers,filters,actions] = [
    ref([{key:"dlq",value:"dlq",text:"A new DLQ message is sent to the DLQ topic."},
      {key:"fail",value:"fail",text:"Consumer message failed"},
      {key:"system",value:"system",text:"Beanq system error"}]),
    ref([{key:"default-channel",value:"default-channel"},{key:"order-channel",value:"order-channel"}]),
    ref([{key:"slack",value:"slack"},{key:"email",value:"email"}])
];

onMounted(()=>{
  list();
})
//
// watch(() => form.value.google.clientId, (n, o) => {
//   let ele = document.getElementById("clientId");
//   if(n !== ""){
//     ele.style.cssText = "border-color: #ced4da;";
//   }else {
//     ele.style.cssText = "border-color: red;";
//   }
// })
//
// watch(()=> form.value.google.clientSecret, (n, o) => {
//   let ele = document.getElementById("clientSecret");
//   if(n !== ""){
//     ele.style.cssText = "border-color: #ced4da;";
//   }else {
//     ele.style.cssText = "border-color: red;";
//   }
// })
//
// watch(()=> form.value.google.callBackUrl, (n, o) => {
//   let ele = document.getElementById("callBackUrl");
//   if(n !== ""){
//     ele.style.cssText = "border-color: #ced4da;";
//   }else {
//     ele.style.cssText = "border-color: red;";
//   }
// })

const onTestNotify = async (param) => {
  console.log(param)
  try {
    let data = {
      smtp:{
        host: form.value.smtp.host,
        port: form.value.smtp.port,
        user: form.value.smtp.user,
        password: form.value.smtp.password,
      },
      sendGrid:{
        key: form.value.grid.key,
        fromName: form.value.grid.fromName,
        fromAddress: form.value.grid.fromAddress
      },
      tools:param,
      slack:{
        botAuthToken: form.value.slack.botAuthToken
      }
    }
    let res = await request.post("/test/notify",data, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    toastRef.value.show("success");
  }catch (e) {
    toastRef.value.show(e);
  }
}

const list = async () => {
  try {
    let res = await configApi.getConfig();
    if(res?.google){
      form.value.google = res.google;
    }
    if(res?.smtp){
      form.value.smtp = res.smtp;
    }
    if(res?.sendGrid){
      form.value.sendGrid = res.sendGrid;
    }
    if(res?.rule){
      let rule = res.rule;
      form.value.rule = rule;
    }
    if(res?.slack){
      form.value.slack = res.slack;
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

const edit = async () => {

  let value = form.value || {};
  // if (value.google.clientId === "") {
  //   document.getElementById("clientId").style.cssText = "border-color: red;";
  //   return;
  // }
  // if(value.google.clientSecret === ""){
  //   document.getElementById("clientSecret").style.cssText = "border-color: red;";
  //   return;
  // }
  // if(value.google.callBackUrl === ""){
  //   document.getElementById("callBackUrl").style.cssText = "border-color: red;";
  //   return;
  // }

  let result = {
    google:{
      clientId: value.google.clientId,
      clientSecret: value.google.clientSecret,
      callBackUrl: value.google.callBackUrl,
      scheme: value.google.scheme
    },
    smtp:{
      host: value.smtp.host,
      port: value.smtp.port,
      user: value.smtp.user,
      password: value.smtp.password,
    },
    sendGrid:{
      key: value.grid.key,
      fromName: value.grid.fromName,
      fromAddress: value.grid.fromAddress
    },
    slack:{
      botAuthToken: value.slack.botAuthToken
    },
    rule:{
      when: value.rule.when,
      if: value.rule.if,
      then: value.rule.then
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
