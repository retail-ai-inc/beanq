<template>
  <div class="modal fade" :id="id" data-bs-keyboard="false" tabindex="-1" :aria-labelledby="label">
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
            <div :for="key" class="col-md-3 col-form-label" style="font-weight: bold" >{{key}}</div>
            <div class="col-md-9" style="display: flex;flex-direction: column;justify-content: center">
              <div v-if="key === 'payload'" :id="key">
                <CodeMirrorEditor :data="item" @getValue="getV"/>
              </div>
              <div v-else :id="key">
                <pre style="margin:0"><code>{{item}}</code></pre>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
          <button type="button" class="btn btn-primary" @click="action">Edit</button>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps,defineEmits,ref} from "vue";
import CodeMirrorEditor from "../../components/CodeMirrorEditor.vue";

const props = defineProps({
  label:"",
  id:"",
  data:{}
})

const [localData] = [ref(null)];

const emits = defineEmits(['action']);
const action = function (){
    emits("action",localData.value);
}

function getV(data){
  localData.value = props.data;
  localData.value.payload = data;
}

</script>
<style scoped>

</style>
