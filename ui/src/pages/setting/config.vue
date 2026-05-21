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
              <button class="nav-link active" id="google-tab" data-bs-toggle="tab" data-bs-target="#google-pan" type="button" role="tab" aria-controls="google-pan" aria-selected="true">Google Login</button>
              <button class="nav-link" id="google-recaptcha-tab" data-bs-toggle="tab" data-bs-target="#google-recaptcha-pan" type="button" role="tab" aria-controls="google-recaptcha-pan" aria-selected="false">Google Recaptcha</button>
              <button class="nav-link" id="smtp-tab" data-bs-toggle="tab" data-bs-target="#smtp-pan" type="button" role="tab" aria-controls="smtp-pan" aria-selected="false">Email</button>
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
            <GoogleRecaptcha class="tab-pane fade"
                     id="google-recaptcha-pan"
                     role="tabpanel"
                     aria-labelledby="google-recaptcha-tab"
                     tabindex="0"
                     v-model="form.googleRecaptcha"
            />
            <!--smtp-->
            <Smtp class="tab-pane fade"
                  id="smtp-pan"
                  role="tabpanel"
                  aria-labelledby="smtp-tab"
                  tabindex="0"
                  v-model="form.email"
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

          </div>
          <div class="d-grid gap-2 col-1 mx-auto">
            <button class="btn btn-primary" type="button" @click="edit">{{$t('save')}}</button>
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
import GoogleRecaptcha from "./config/googleRecaptcha.vue";
import Smtp from "./config/smtp.vue";
import SendGrid from "./config/sendGrid.vue";
import AlertRule from "./config/alertRule.vue";
import Slack from "./config/slack.vue";

const [noticeId,loginModal] = [ref("configBackdrop"),ref("loginModal")];
const [toastId,toastRef] = [ref("toast-" + Math.random().toString(36)),ref("toastRef")]
const googleCallback = `${window.location.origin}/callback`;

const form = ref({
  google:{
    clientId: "",
    clientSecret: "",
    callBackUrl: googleCallback,
  },
  googleRecaptcha:{
    active:false,
    projectId:"",
    siteKeyV2:"",
    siteKeyV3:"",
    apiKey:""
  },
  email:{
    used:"smtp",
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

const onTestNotify = async (param) => {

  try {
    let {used,smtp,sendGrid:grid} = form.value.email;

    let data = {
      used:used,
      smtp:{
        host: smtp.host,
        port: smtp.port,
        user: smtp.user,
        password: smtp.password,
      },
      sendGrid:{
        key: grid.key,
        fromName: grid.fromName,
        fromAddress: grid.fromAddress
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
      if(res.google.callBackUrl === ""){
        res.google.callBackUrl = googleCallback;
      }
      form.value.google = res.google;
    }
    if(res?.googleReCAPTCHA){
      form.value.googleRecaptcha = res.googleReCAPTCHA;
    }
    form.value.email.used = res?.email?.used || "smtp";
    if(res?.email.smtp){
      form.value.email.smtp = res.email.smtp;
    }
    if(res?.email.sendGrid){
      form.value.email.sendGrid = res.email.sendGrid;
    }
    if(res?.rule){
      let rule = res.rule;
      form.value.rule = rule;
    }
    if(res?.slack){
      form.value.slack = res.slack;
    }
console.log(form.value)
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

  let result = {
    google:{
      clientId: value.google.clientId,
      clientSecret: value.google.clientSecret,
      callBackUrl: form.value.google.callBackUrl,
    },
    googleRecaptcha:{
      active:value.googleRecaptcha.active,
      projectId:value.googleRecaptcha.projectId,
      siteKeyV2:value.googleRecaptcha.siteKeyV2,
      siteKeyV3:value.googleRecaptcha.siteKeyV3,
      apiKey:value.googleRecaptcha.apiKey
    },
    email:{
      used:value.email.used,
      smtp:{
        host: value.email.smtp.host,
        port: value.email.smtp.port,
        user: value.email.smtp.user,
        password: value.email.smtp.password,
      },
      sendGrid:{
        key: value.email.sendGrid.key,
        fromName: value.email.sendGrid.fromName,
        fromAddress: value.email.sendGrid.fromAddress
      }
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
