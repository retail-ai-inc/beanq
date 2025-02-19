<template>
  <div class="row mb-4">
    <div class="col">
      <h5 class="card-title">List of Event Log</h5>
    </div>
  </div>
  <div class="row">
    <div class="col-12">
      <div class="row">
        <div class="col">
          <div class="form-row mb-3">
            <div class="col-lg-2 col-sm-8" style="padding-right: 10px;">
              <div class="form-row">
                <div class="col">
                  <select class="form-select" aria-label="Default select" id="formStatus" name="formStatus" style="cursor: pointer" v-model="form.status">
                    <option selected value="">All status</option>
                    <option value="published">Published</option>
                    <option value="success">Success</option>
                    <option value="failed">Failed</option>
                  </select>
                </div>
              </div>
            </div>
            <div class="col" style="padding-right: 10px;">
              <input type="text" class="form-control" id="formId" name="formId"  v-model="form.id" placeholder="Search by Id">
            </div>
            <div class="col-auto">
              <button type="submit" class="btn btn-primary" @click="search">{{searchBtn}}</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {ref,inject,defineProps,watch,defineEmits} from "vue";

const l = inject("i18n");

const props = defineProps({
  form:{
    id:"",
    status:""
  }
})
const btns = ref(OtherBtn);
const searchBtn = ref(roleApi.GetLang("Search",btns.value)?.[l.value]);
watch(()=>[l.value],([n,o])=>{
  searchBtn.value = roleApi.GetLang("Search",btns.value)?.[n];
})

const emits = defineEmits(['search']);
const search = function (){
  emits("search");
}
</script>

<style>
.form-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}
</style>