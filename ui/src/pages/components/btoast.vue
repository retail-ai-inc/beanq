<template>
  <div class="toast-container position-fixed top-0 end-0 p-3">
    <div :id="id" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
      <div class="progress" role="progressbar"
           aria-label="progress 1px high"
           aria-valuenow="25" aria-valuemin="0" aria-valuemax="100" style="height: 5px">
          <div class="progress-bar bg-success" :style="{'width': progress + '%'}"></div>
      </div>
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
import {ref,defineProps,defineExpose,onUnmounted} from "vue";
const props = defineProps({
  id:"",
})
const body = ref("");
const progress = ref(100);
let timer = null;

let animationFrameId = null;

function show(err){

  const myToastEl = document.getElementById(props.id);
  const m = new bootstrap.Toast(myToastEl);
  
  body.value = err;
  m.show();

  const startTime = performance.now();
  const duration = 5000;
  
  function animate(currentTime) {
    const elapsed = currentTime - startTime;
    const percentRemaining = Math.max(0, 100 - (elapsed / duration * 100));
    progress.value = percentRemaining;
    
    if (percentRemaining > 0) {
      animationFrameId = window.requestAnimationFrame(animate);
    } else {
      m.hide();
      animationFrameId = null;
      myToastEl.addEventListener('hidden.bs.toast', () => {
        progress.value = 100;
      })
    }
  }

  animationFrameId = window.requestAnimationFrame(animate);
}

onUnmounted(()=>{
  if (animationFrameId !== null) {
    window.cancelAnimationFrame(animationFrameId);
  }
})

defineExpose({
  show
})
</script>