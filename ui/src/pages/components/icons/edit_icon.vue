<template>
  <a class="btn btn-primary" href="javascript:;" @click="action"
     data-bs-toggle="tooltip"
     data-bs-placement="top"
     :data-bs-title="title"
     ref="editRef"
     >
    <div class="icon-button">
      <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-pencil-square" viewBox="0 0 16 16">
        <path d="M15.502 1.94a.5.5 0 0 1 0 .706L14.459 3.69l-2-2L13.502.646a.5.5 0 0 1 .707 0l1.293 1.293zm-1.75 2.456-2-2L4.939 9.21a.5.5 0 0 0-.121.196l-.805 2.414a.25.25 0 0 0 .316.316l2.414-.805a.5.5 0 0 0 .196-.12l6.813-6.814z"/>
        <path fill-rule="evenodd" d="M1 13.5A1.5 1.5 0 0 0 2.5 15h11a1.5 1.5 0 0 0 1.5-1.5v-6a.5.5 0 0 0-1 0v6a.5.5 0 0 1-.5.5h-11a.5.5 0 0 1-.5-.5v-11a.5.5 0 0 1 .5-.5H9a.5.5 0 0 0 0-1H2.5A1.5 1.5 0 0 0 1 2.5v11z"/>
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
    default:"Edit"
  }
})

const [editRef] = [ref("editRef")];
onMounted(()=>{
  new bootstrap.Tooltip(editRef.value);
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