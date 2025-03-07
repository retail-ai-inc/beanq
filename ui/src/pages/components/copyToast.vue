<template>
  <div class="toast-container position-fixed top-0 end-50 p-3">
    <div  :id="id" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
      <div class="toast-body" style="padding: 0 !important;word-wrap:normal">
        <div class="alert alert-success" role="alert" style="margin: 0 !important;">
          Copy to clipboard!
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps,defineExpose} from "vue";
const props = defineProps({
  id:"",
})
function show(err){
  const myToastEl = document.getElementById(props.id);
  let ntoast = new bootstrap.Toast(myToastEl,{
    delay:2000
  });
  ntoast.show();
}
defineExpose({
  show
})
</script>