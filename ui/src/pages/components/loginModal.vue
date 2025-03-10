<template>
  <div>
    <div class="modal fade" :id="id" data-bs-backdrop="static" data-bs-keyboard="false" tabindex="-1" aria-labelledby="staticBackdropLabel" aria-hidden="true">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h1 class="modal-title fs-5" id="staticBackdropLabel">Login has expired</h1>
          </div>
          <div class="modal-body" style="padding: 2.5rem 0;text-align: center">
            <button class="btn btn-primary" @click="reLogin">ReLogin</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {ref,defineProps,defineExpose,onUnmounted,onMounted} from "vue";
import { useRouter } from 'vueRouter';

const props = defineProps({
  id:"",
})

const [noticeModal] = [ref("staticBackdrop")];
const showNoticeModal = (()=>{
  const eleRetry = document.getElementById(props.id);
  noticeModal.value = new bootstrap.Modal(eleRetry);
  noticeModal.value.show(eleRetry);
})

const [uRouter] = [useRouter()];
const reLogin=(()=>{
  Storage.Clear();
  uRouter.replace("/login");
})
onMounted(()=>{
  const eleRetry = document.getElementById(props.id);
  noticeModal.value = new bootstrap.Modal(eleRetry);
})
onUnmounted(()=>{
  noticeModal.value.dispose();
})

function error(err){
  showNoticeModal();
}
defineExpose({
  error
})
</script>