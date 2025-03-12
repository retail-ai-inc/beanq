<template>
  <span
     data-bs-toggle="tooltip"
     data-bs-placement="top"
     :data-bs-title="pastTime"
     ref="timeRef">
    {{formatRelativeTime(pastTime)}}
  </span>
</template>
<script setup>
import {ref,defineProps,defineEmits,onMounted} from "vue";
const props = defineProps({
  pastTime:{
    type:String,
    required:true,
    default:""
  }
})
const [timeRef] = [ref(null)];
onMounted(()=>{
  new bootstrap.Tooltip(timeRef.value);
})

const formatRelativeTime = ((pastTime)=>{
  return Base.FormatRelativeTime(pastTime);
})
</script>
<style scoped>

</style>