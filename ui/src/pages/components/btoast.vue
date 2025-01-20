<template>
  <div class="toast-container position-fixed bottom-0 end-0 p-3">
    <div :id="id" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
      <div class="toast-header">
        <strong class="me-auto">
          <slot name="title">
            Notice
          </slot>
        </strong>
        <button type="button" class="btn-close" data-bs-dismiss="toast" aria-label="Close"></button>
      </div>
      <div class="toast-body">
        <slot name="body">
          {{body}}
        </slot>
      </div>
    </div>
  </div>
</template>
<script setup>
import {ref,defineProps,defineExpose} from "vue";
const props = defineProps({
  id:"",
})
const body = ref("");

function show(err){
  const myToastEl = document.getElementById(props.id);
  const m = new bootstrap.Toast(myToastEl);
  body.value = err;
  m.show()
}
defineExpose({
  show
})
</script>