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
              <button class="nav-link" id="alert-rule-tab" data-bs-toggle="tab" data-bs-target="#alert-rule-pane" type="button" role="tab" aria-controls="alert-rule-pane" aria-selected="false">Alert Rule</button>
            </div>
          </nav>


          <div class="tab-content" id="myTabContent">
            <!--google credential-->
            <div class="tab-pane fade show active" id="google-pan" role="tabpanel" aria-labelledby="google-tab" tabindex="0">
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
              <button type="button" class="btn btn-primary" @click="edit" style="margin-top: 2rem;">{{$t('edit')}}</button>
            </div>
            <!--smtp-->
            <div class="tab-pane fade" id="smtp-pan" role="tabpanel" aria-labelledby="smtp-tab" tabindex="0">
              <div class="container">
                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="clientId" class="col-form-label">Host:</label>
                  </div>
                  <div class="col-6">
                    <input type="text" id="host" class="form-control" v-model="form.smtp.host" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="clientIdHelpInline" class="form-text">

                    </span>
                  </div>
                </div>

                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="clientSecret" class="col-form-label">Port:</label>
                  </div>
                  <div class="col-6">
                    <input type="text" id="port" class="form-control" v-model="form.smtp.port" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="clientSecretHelpInline" class="form-text">

                    </span>
                  </div>
                </div>

                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="callBackUrl" class="col-form-label">User:</label>
                  </div>
                  <div class="col-6">
                    <input type="text" id="user" class="form-control" v-model="form.smtp.user" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="callBackUrlHelpInline" class="form-text">

                    </span>
                  </div>
                </div>
                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="callBackUrl" class="col-form-label">Password:</label>
                  </div>
                  <div class="col-6">
                      <input type="text" id="password" class="form-control" v-model="form.smtp.password" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="callBackUrlHelpInline" class="form-text">

                    </span>
                  </div>
                </div>

              </div>
              <button type="button" class="btn btn-primary" @click="edit" style="margin-top: 2rem;">{{$t('edit')}}</button>
            </div>
            <!--send grid-->
            <div class="tab-pane fade" id="send-grid-pane" role="tabpanel" aria-labelledby="send-grid-tab" tabindex="0">
              <div class="container">
                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="clientId" class="col-form-label">Key:</label>
                  </div>
                  <div class="col-6">
                    <input type="text" id="key" class="form-control" v-model="form.grid.key" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="clientIdHelpInline" class="form-text">

                    </span>
                  </div>
                </div>

                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="clientSecret" class="col-form-label">FromName:</label>
                  </div>
                  <div class="col-6">
                    <input type="text" id="fromName" class="form-control" v-model="form.grid.fromName" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="clientSecretHelpInline" class="form-text">

                    </span>
                  </div>
                </div>

                <div class="row g-3 align-items-center m-2">
                  <div class="col-1 text-end">
                    <label for="callBackUrl" class="col-form-label">FromAddress:</label>
                  </div>
                  <div class="col-6">
                    <input type="text" id="fromAddress" class="form-control" v-model="form.grid.fromAddress" aria-describedby="passwordHelpInline" required>
                  </div>
                  <div class="col-3">
                    <span id="callBackUrlHelpInline" class="form-text">

                    </span>
                  </div>
                </div>

              </div>
              <button type="button" class="btn btn-primary" @click="edit" style="margin-top: 2rem;">{{$t('edit')}}</button>
            </div>
            <!--alert rule-->
            <div class="tab-pane fade" id="alert-rule-pane" role="tabpanel" aria-labelledby="rule-tab" tabindex="0">
              <div class="container">
                <div class="row g-3 align-items-center m-2">
                  <div class="col">
                    <div>
                      <span style="font-weight: bold;color:#0d6efd">WHEN:</span>
                      <span style="font-size: .95rem"> an event is captured by Beanq and all of the following happens</span>
                    </div>

                    <div class="d-flex justify-content-between"
                         style="margin:.5rem 0;background: #fbfbfc;border-radius: .3rem;padding: .5rem;border: 1px solid #ccc;"
                         v-for="(item,index) in triggerList"
                         :key="index"
                    >
                      <div>{{item.text}}</div>
                      <div>
                        <Delete_icon @click="deleteTrigger(item)" style="cursor: pointer"/>
                      </div>
                    </div>
                    <div class="dropdown">
                      <a class="btn btn-secondary dropdown-toggle"
                         href="javascript:;" role="button"
                         data-bs-toggle="dropdown"
                         aria-expanded="false"
                         style="width: 100%;display: flex;justify-content: space-between;align-items: center;"
                      >
                        Add New Trigger
                      </a>

                      <ul class="dropdown-menu" style="width: 100%">
                        <li v-for="(item,index) in triggers" :key="index" @click="addTrigger(item)">
                          <a class="dropdown-item" href="javascript:;">{{item.value}}</a>
                        </li>
                      </ul>
                    </div>
                  </div>
                </div>
                <div class="row g-3 align-items-center m-2">
                  <div class="col">
                    <div>
                      <span style="font-weight: bold;color:#0d6efd">IF:</span>
                      <span style="font-size: .95rem">these filters match</span>
                    </div>
                    <div class="d-flex justify-content-between"
                         style="margin:.5rem 0;background: #fbfbfc;border-radius: .3rem;padding: .5rem;border: 1px solid #ccc;"
                         v-for="(item,index) in filterList" :key="index"
                    >
                      <div>{{item.value}}</div>
                      <div>
                        <Delete_icon @click="deleteFilter(item)" style="cursor: pointer"/>
                      </div>
                    </div>

                    <div class="dropdown">
                      <a class="btn btn-secondary dropdown-toggle"
                         href="javascript:;" role="button"
                         data-bs-toggle="dropdown"
                         aria-expanded="false"
                         style="width: 100%;display: flex;justify-content: space-between;align-items: center;"
                      >
                        Add New Filter
                      </a>

                      <ul class="dropdown-menu" style="width: 100%">
                        <li v-for="(item,index) in filters" :key="index" @click="addFilter(item)">
                          <a class="dropdown-item" href="javascript:;">{{item.value}}</a>
                        </li>
                      </ul>
                    </div>

                  </div>
                </div>
                <div class="row g-3 align-items-center m-2">
                  <div class="col">
                    <div>
                      <span style="font-weight: bold;color:#0d6efd">THEN:</span>
                      <span style="font-size: .95rem">perform these actions</span>
                    </div>
                    <div class="d-flex justify-content-between"
                         style="margin:.5rem 0;background: #fbfbfc;border-radius: .3rem;padding: .5rem;border: 1px solid #ccc;"
                         v-for="(item,index) in actionList" :key="index"
                    >
                      <div>{{item.value}}</div>
                      <div>
                        <Delete_icon @click="deleteAction(item)" style="cursor: pointer"/>
                      </div>
                    </div>
                    <div class="dropdown">
                      <a class="btn btn-secondary dropdown-toggle"
                         href="javascript:;" role="button"
                         data-bs-toggle="dropdown"
                         aria-expanded="false"
                         style="width: 100%;display: flex;justify-content: space-between;align-items: center;"
                      >
                        Add Action
                      </a>

                      <ul class="dropdown-menu" style="width: 100%">
                        <li v-for="(item,index) in actions" :key="index" @click="addAction(item)">
                          <a class="dropdown-item" href="javascript:;">{{item.value}}</a>
                        </li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
              <button type="button" class="btn btn-primary" @click="edit" style="margin-top: 2rem;">{{$t('edit')}}</button>
            </div>
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
  grid:{
    key:"",
    fromName:"",
    fromAddress:""
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
const [triggerList,filterList,actionList] = [ref([]),ref([]),ref([])];

onMounted(()=>{
  list();
})

const addItem= (arr,item) => {
  if(!Array.isArray(arr)){
    return [];
  }
  if(arr.find(i => i.key === item.key)){
    return arr;
  }
  arr.push({
    key: item.key,
    value: item.value,
    text: item.text
  });
  return arr;
}

const deleteItem= (arr,item) => {
  if(!Array.isArray(arr)){
    return [];
  }
  arr = arr.filter(i => i.key !== item.key);
  return arr;
}

const addTrigger= (item) => {
  triggerList.value = addItem(triggerList.value,item);
}
const deleteTrigger = (item) => {
  triggerList.value = deleteItem(triggerList.value,item);
}
const addFilter= (item) => {
  filterList.value = addItem(filterList.value,item);
}
const deleteFilter = (item) => {
  filterList.value = deleteItem(filterList.value,item);
}
const addAction= (item) => {
  actionList.value = addItem(actionList.value,item);
}
const deleteAction = (item) => {
  actionList.value = deleteItem(actionList.value,item);
}

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
    if(res?.google){
      form.value.google = JSON.parse(res.google);
    }
    if(res?.smtp){
      form.value.smtp = JSON.parse(res.smtp);
    }
    if(res?.sendGrid){
      form.value.sendGrid = JSON.parse(res.sendGrid);
    }
    if(res?.rule){
      let rule = JSON.parse(res.rule);
      triggerList.value = rule?.when;
      filterList.value = rule?.if;
      actionList.value = rule?.then;
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
    },
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
    rule:{
      when: triggerList.value,
      if: filterList.value,
      then: actionList.value
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