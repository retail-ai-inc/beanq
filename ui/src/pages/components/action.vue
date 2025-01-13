<template>
  <div class="modal fade" data-bs-keyboard="false" tabindex="-1" :aria-labelledby="label" aria-hidden="true" :id="id">
    <div class="modal-dialog modal-md modal-dialog-centered">
      <div class="modal-content">
        <div class="modal-header">
          <h1 class="modal-title fs-5" :id="label">
            <slot name="title">Are you sure to retry?</slot></h1>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
        </div>
        <div class="modal-body">
          <p><slot name="body">Trying again will not restore the data</slot></p>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-light" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-danger" @click="action">Yes</button>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps,defineEmits} from "vue";

const props = defineProps({
  label:"",
  id:"",
})

const emits = defineEmits(['action']);
const action = function (){
  emits("action");
}
</script>
