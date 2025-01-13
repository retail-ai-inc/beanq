<template>
  <div class="modal fade" :id="id" data-bs-keyboard="false" tabindex="-1" :aria-labelledby="label" aria-hidden="true">
    <div class="modal-dialog modal-lg">
      <div class="modal-content">
        <div class="modal-header">
          <h1 class="modal-title fs-5" :id="label">
            <slot name="title">Edit Payload</slot>
          </h1>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
        </div>
        <div class="modal-body">
          <div class="mb-3 row" v-for="(item,key) in data" :key="key">
            <label :for="key" class="col-sm-2 col-form-label" style="font-weight: bold">{{key}}</label>
            <div class="col-sm-10">

              <div id="payloadAlertInfo" v-if="key === 'payload'">
              </div>

              <textarea class="form-control" id="payload" rows="3" v-if="key === 'payload'" v-model="data.payload" @blur="payloadTrigger"></textarea>
              <input type="text" readonly :id="key" class="form-control-plaintext" :value="item" v-else>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
          <button type="button" class="btn btn-primary" @click="action(data)">Edit</button>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps,defineEmits,ref} from "vue";

const props = defineProps({
  label:"",
  id:"",
  data:{}
})

const isFormat = ref(false);

const emits = defineEmits(['action']);
const action = function (obj){
  emits("action",obj);
}

async function payloadTrigger(){

  isFormat.value = false;
  try {
    await JSON.parse(props.data.payload);
  }catch (e) {
    isFormat.value = true;
  }
  if (isFormat.value === true){
    await Alert("Must be in JSON format","danger");
    return;
  }
  const alertTrigger = new bootstrap.Alert('#my-alert');
  alertTrigger.close();

}

function Alert(message,type){
  const alertPlaceholder = document.getElementById('payloadAlertInfo');
  alertPlaceholder.innerHTML = `<div class="alert alert-${type} alert-dismissible" id="my-alert" role="alert">
      <div>${message}</div>
      <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
      </div>`;
}

</script>
