<template>
  <div class="modal fade" data-bs-keyboard="false" tabindex="-1" :aria-labelledby="label" :id="id">
    <div class="modal-dialog modal-md modal-dialog-centered">
      <div class="modal-content">
        <div class="modal-header">
          <h1 class="modal-title fs-5" :id="label">
            <slot name="title">Are you sure to retry?</slot></h1>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
        </div>
        <div class="modal-body">
          <div class="alert alert-danger" role="alert">
            <div v-html="warning" style="font-weight: bold"></div>
            <hr/>
            <div v-html="info"></div>
            <b>{{dataId}}</b>
          </div>
          <input
              type="text"
              :id="`input-${id}`"
              class="form-control"
              v-model="dataIdValue"
              placeholder="Please enter the prompt content to continue."
              @input="checkInput"
          />
          <div ref="notice" class="notice" style="color: #b02a37;margin-top: .35rem;">
            {{noticeMsg}}
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-light" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-danger" @click="action" :disabled="disable">Yes</button>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {ref,defineProps,defineEmits} from "vue";

const props = defineProps({
  label:"",
  id:"",
  dataId:"",
  warning:{
    type:String,
    default:"Warning: Item deletion cannot be undone!<br/> Please proceed with caution!"
  },
  info:{
    type:String,
    default:"This operation will permanently delete the data of log.<br>\n" +
        "To prevent accidental actions, please confirm by entering the following:<br/>"
  }
})

const [dataIdValue,disable] = [ref(""),ref(true)];
const [noticeMsg] = [ref("")];
const checkInput = Base.Debounce (()=>{
  if(dataIdValue.value.length <= 0){
    noticeMsg.value = "";
    disable.value = true;
  }else{
    disable.value = false;
  }
},300);

const emits = defineEmits(['action']);
const action = function (){

  if(dataIdValue.value.length <= 0){
    noticeMsg.value = "";
    return;
  }
  if(dataIdValue.value !== props.dataId){
    noticeMsg.value = "ID mismatch";
    return;
  }
  emits("action");
}
</script>
