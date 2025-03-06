<template>
  <div class="d-flex">
    <span class="d-block text-truncate" style="max-width: 8rem;">{{payload}}</span>
    <a tabindex="0"
       class="link-primary"
       role="button"
       data-bs-toggle="popover"
       data-bs-trigger="focus"
       data-bs-placement="top"
       data-bs-custom-class="custom-popover"
       ref="popoverRef"
       style="font-size: 0.9rem;">more</a>
  </div>
</template>
<script setup>
import {ref,defineProps,defineEmits,onMounted,onUnmounted} from "vue";
const props = defineProps({
  payload:{
    type:String,
    required:true,
    default:""
  },
})

const popoverRef = ref(null);

onMounted(()=>{
  let str = JSON.stringify(JSON.parse(props.payload),null,2);
  let payload = `<pre><code>${str}</code></pre>`;
  new bootstrap.Popover(popoverRef.value,{
    html:true,
    trigger:"focus",
    title:"payload",
    content:payload
  })
})

onUnmounted(()=>{

})

</script>
<style scoped></style>