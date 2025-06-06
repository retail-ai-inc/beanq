<template>
  <a class="btn btn-danger" href="javascript:;" @click="action"
     data-bs-toggle="tooltip"
     data-bs-placement="top"
     :data-bs-title="title"
     ref="deleteRef">
  <div class="icon-button">
    <svg xmlns="http://www.w3.org/2000/svg" width="100%" height="100%" fill="currentColor" class="bi bi-unlock2-fill" viewBox="0 0 16 16">
      <path fill-rule="evenodd" d="M8 0c1.07 0 2.041.42 2.759 1.104l.14.14.062.08a.5.5 0 0 1-.71.675l-.076-.066-.216-.205A3 3 0 0 0 5 4v2h6.5A2.5 2.5 0 0 1 14 8.5v5a2.5 2.5 0 0 1-2.5 2.5h-7A2.5 2.5 0 0 1 2 13.5v-5a2.5 2.5 0 0 1 2-2.45V4a4 4 0 0 1 4-4"/>
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
    default:"Delete"
  }
})

const [deleteRef] = [ref(null)];
onMounted(()=>{
  new bootstrap.Tooltip(deleteRef.value);
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
