<template>
  <div class="row">

    <div class="col-md-2">
      <div class="row">
        <label for="formId" class="col-md-4 col-form-label text-end">Id:</label>
        <div class="col-md-8">
          <input type="text" class="form-control" id="formId" name="formId"  v-model="form.id">
        </div>
      </div>
    </div>

    <div class="col-md-3">
      <div class="row">
        <label for="formStatus" class="col-md-4 col-form-label text-end">Status:</label>
        <div class="col-md-8">
          <select class="form-select" aria-label="Default select" id="formStatus" name="formStatus" style="cursor: pointer" v-model="form.status">
            <option selected value="">Open this select</option>
            <option value="published">Published</option>
            <option value="success">Success</option>
            <option value="failed">Failed</option>
          </select>
        </div>
      </div>
    </div>

    <div class="col-2">
      <div class="col-auto">
        <button type="submit" class="btn btn-primary mb-3" @click="search">Search</button>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps,defineEmits} from "vue";

const props = defineProps({
  form:{
    id:"",
    status:""
  }
})

const emits = defineEmits(['search']);
const search = function (){
  emits("search");
}
</script>