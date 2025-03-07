<template>
  <div>
    <div @click="copyText(text)" style="cursor: pointer">
      {{maskString(text)}}
    </div>
    <CopyToast :id="copyToast" ref="copyRef"/>
  </div>

</template>
<script setup>
import {ref,defineProps} from "vue";
import CopyToast from "./copyToast.vue";
const props = defineProps({
  text:{
    type:String,
    required:true,
    default:""
  },
})

const maskString = ((id)=>{
  return Base.MaskString(id)
})

const [copyToast,copyRef] = [ref("copyToast"),ref("copyRef")];
const copyText = (async (text)=>{
  try {
    await navigator.clipboard.writeText(text);
    copyRef.value.show();
  } catch (err) {
    console.error('copied error:', err);
  }
})

</script>
<style scoped>

</style>