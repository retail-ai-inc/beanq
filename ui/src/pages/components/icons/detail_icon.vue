<template>
  <a class="btn btn-success" href="javascript:;" @click="action"
     data-bs-toggle="tooltip"
     data-bs-placement="top"
     :data-bs-title="title"
     ref="detailRef">
    <div class="icon-button">
      <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-eye-fill" viewBox="0 0 16 16">
        <path d="M10.5 8a2.5 2.5 0 1 1-5 0 2.5 2.5 0 0 1 5 0"/>
        <path d="M0 8s3-5.5 8-5.5S16 8 16 8s-3 5.5-8 5.5S0 8 0 8m8 3.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7"/>
      </svg>
    </div>
  </a>
</template>
<script setup>
import {ref,defineProps,defineEmits,onMounted} from "vue";

const props = defineProps({
  item:{},
  title:{
    type:String,
    default:"View"
  }
})

const [detailRef] = [ref(null)];
onMounted(()=>{
  new bootstrap.Tooltip(detailRef.value);
})

const emits = defineEmits(['action']);
const action = function (){
  emits("action",props.item);
}
</script>
<style scoped>
.btn{
  padding:.245rem .45rem;
  .icon-button{
    width: 1.125rem;
    height: 1.425rem;
  }
}
</style>