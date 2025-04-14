<template>
  <div class="tab-pane fade" id="alert-rule-pane" role="tabpanel" aria-labelledby="rule-tab" tabindex="0">
    <div class="container">
      <div class="row g-3 align-items-center m-2">
        <div class="col">
          <div style="padding: .3rem 0">
            <span class="cond-pre">WHEN:</span>
            <span style="font-size: .95rem"> an event is captured by Beanq and all of the following happens</span>
          </div>

          <div class="d-flex justify-content-between align-items-center m-1 p-2 rounded-2"
               style="background: #fbfbfc;border: 1px solid #ccc;"
               v-for="(item,index) in rules.when"
               :key="index"
          >
            <div>{{item.text}}</div>
            <div>
              <Delete_icon @click="deleteTrigger(item)" style="cursor: pointer"/>
            </div>
          </div>
          <div class="dropdown">
            <a class="btn btn-primary dropdown-toggle d-flex justify-content-between align-items-center"
               href="javascript:;" role="button"
               data-bs-toggle="dropdown"
               aria-expanded="false"
               style="width: 100%;"
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
          <div style="padding: .3rem 0">
            <span class="cond-pre">IF:</span>
            <span style="font-size: .95rem">these filters match</span>
          </div>
          <div class="d-flex justify-content-between  align-items-center m-1 p-2 rounded-2"
               style="background: #fbfbfc;border: 1px solid #ccc;"
               v-for="(item,index) in rules.if" :key="index"
          >
            <div>
              {{item.value}}:<span v-for="(it,int) in item.topic" :key="int">{{it.topic}},</span>
            </div>
            <div>
              <Delete_icon @click="deleteFilter(item)" style="cursor: pointer"/>
            </div>
          </div>
          <div>

            <div class="dropdown">
              <a class="btn btn-primary dropdown-toggle d-flex justify-content-between align-items-center"
                 href="javascript:;" role="button"
                 data-bs-toggle="dropdown"
                 aria-expanded="false"
                 style="width: 100%;"
              >
                Add New Filter
              </a>

              <ul class="dropdown-menu" style="width: 100%">
                <li v-for="(item,index) in filters" :key="index" @click="addFilter(item)">
                  <a class="dropdown-item" href="javascript:;">{{item.value}}</a>
                </li>
              </ul>
            </div>

            <div class="dropdown" style="margin-top: .3rem">
              <a class="btn btn-primary dropdown-toggle d-flex justify-content-between align-items-center"
                 href="javascript:;" role="button"
                 data-bs-toggle="dropdown"
                 aria-expanded="false"
                 style="width: 100%;"
              >
                Choose Topic
              </a>

              <ul class="dropdown-menu" style="width: 100%">
                <li v-for="(item,index) in nchannel" :key="index" @click="addTopic(item)">
                  <a class="dropdown-item" href="javascript:;">{{item.topic}}</a>
                </li>
              </ul>
            </div>
          </div>

        </div>
      </div>
      <div class="row g-3 align-items-center m-2">
        <div class="col">
          <div style="padding: .3rem 0">
            <span class="cond-pre">THEN:</span>
            <span style="font-size: .95rem">perform these actions</span>
          </div>
          <div class="d-flex justify-content-between align-items-center m-1 p-2 rounded-2"
               style="background: #fbfbfc;border: 1px solid #ccc;"
               v-for="(item,index) in rules.then" :key="index"
          >
            <div class="container">

              <div class="row" v-if="item.key==='email'">
                <label for="emailInput" class="col-2 col-form-label text-end">{{item.key}} address:</label>
                <div class="col-8">
                  <input type="text" class="form-control" id="emailInput" v-model="item.value" placeholder="please input an email"/>
                </div>
              </div>
              <div class="row" v-if="item.key==='slack'">
                <div class="d-flex">
                  <span class="d-flex align-items-center" style="white-space: nowrap">Send a notification to the</span>
                  <span  class="d-flex align-items-center">
                    <select id="disabledSelect" class="form-select" v-model="item.parameters.workSpace">
                      <option :selected="item.parameters.workSpace === 'Retail AI Inc'">Retail AI Inc</option>
                    </select>
                  </span>
                  <span  class="d-flex align-items-center" style="white-space: nowrap">Slack workspace to</span>
                  <span  class="d-flex align-items-center"><input type="text" v-model="item.parameters.channel" class="form-control"/></span>
                  <span  class="d-flex align-items-center" style="white-space: nowrap">and show tags</span>
                  <span  class="d-flex align-items-center"><input type="text" v-model="item.parameters.tags" class="form-control"/></span>
                  <span  class="d-flex align-items-center" style="white-space: nowrap">and notes</span>
                  <span  class="d-flex align-items-center"><input type="text" v-model="item.parameters.notes"  class="form-control"/></span>
                  <span  class="d-flex align-items-center" style="white-space: nowrap">in notification</span>
                </div>
<!--                <label for="slackInput" class="col-2 col-form-label text-end">{{item.key}} webhook:</label>-->
<!--                <div class="col-8">-->
<!--                  <input type="text" class="form-control" id="slackInput" v-model="item.value" placeholder="please input a slack webhook"/>-->
<!--                </div>-->
              </div>

            </div>

            <div>
              <Delete_icon @click="deleteAction(item)" style="cursor: pointer"/>
            </div>
          </div>
          <div class="dropdown">
            <a class="btn btn-primary dropdown-toggle d-flex justify-content-between align-items-center"
               href="javascript:;" role="button"
               data-bs-toggle="dropdown"
               aria-expanded="false"
               style="width: 100%;"
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
      <button type="button" class="btn btn-success m-3" @click="onTestNotify" style="margin-top: 2rem;">Send Test Notification</button>
    </div>
    <Btoast :id="toastId" ref="toastRef"></Btoast>
  </div>
</template>
<script setup>
import { ref,defineProps,computed,defineEmits,onMounted } from "vue";
import Delete_icon from "../../components/icons/delete_icon.vue";
import Btoast from "../../components/btoast.vue";

const props = defineProps({
  modelValue:{
    type:Object,
    required: true,
  }
})
const [toastId,toastRef] = [ref("toast-" + Math.random().toString(36)),ref("toastRef")];

const emit = defineEmits(['update:modelValue','onTestNotify']);
const rules = computed({
  get() {
    return props.modelValue;
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

const [triggers,filters,actions] = [
  ref([{key:"dlq",value:"dlq",text:"A new DLQ message is sent to the DLQ topic."},
    {key:"fail",value:"fail",text:"Consumer message failed"},
    {key:"system",value:"system",text:"Beanq system error"}]),
  ref([]),
  ref([{key:"slack",value:"slack"},{key:"email",value:"email"}])
];

const [channel,nchannel] = [ref([]),ref([])];
const channels = (async()=>{
  try {
    let data = await request.get("queue/list",{"params":{"page":0,"pageSize":100}});
    channel.value = data;
    Object.entries(data).forEach(([key,value]) => {
      filters.value.push({
        key: key,
        value: key
      });
    });
  }catch (err) {
    toastRef.value.show(err);
  }
})

onMounted(()=>{
  channels();
})

const onTestNotify = async()=>{
  emit('onTestNotify',rules.value.then);
}

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
    text: item?.text | "",
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
  rules.value.when = addItem(rules.value.when,item);
  emit('update:modelValue', rules.value);

}
const deleteTrigger = (item) => {
  rules.value.when = deleteItem(rules.value.when,item);
  emit('update:modelValue',rules.value);
}
const addFilter= (item) => {
  rules.value.if = addItem(rules.value.if,item);
  emit('update:modelValue',rules.value);
  nchannel.value = channel.value[item.key];
}

const addTopic= (item) => {

  rules.value.if.forEach((i) => {
    if(i.key === item.channel){
      if(!i?.topic){
        i.topic = [item];
        return;
      }
      i.topic.forEach((t) => {
        if(t.topic !== item.topic){
          i.topic.push(item);
        }
      })
    }
  })
  console.log("new if:",rules.value.if);
}

const deleteFilter = (item) => {
  rules.value.if = deleteItem(rules.value.if,item);
  emit('update:modelValue',rules.value);
}
const addAction= (item) => {
  rules.value.then = addItem(rules.value.then,item);
  console.log(rules.value.then);
  return
  emit('update:modelValue',rules.value);
}
const deleteAction = (item) => {
  rules.value.then = deleteItem(rules.value.then,item);
  emit('update:modelValue',rules.value);
}

</script>
<style scoped>
.cond-pre{
  font-weight: bold;
  color:#fff;
  background-color:#146c43;
  padding:.3rem 1rem;
  border-radius: .2rem
}
.m-2{
  margin:2rem .5rem !important;
}
</style>