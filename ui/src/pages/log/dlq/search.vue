<template>
  <div class="row">
    <div class="col-12">
      <div class="row">
        <div class="col">
          <div class="form-row mb-3">
            <div class="col-2" style="padding-right: 1rem">
              <select class="form-select" aria-label="Default select" id="formMoodType" name="formMoodType" style="cursor: pointer" v-model="form.moodType">
                <option selected value="">Mood Type</option>
                <option value="normal">normal</option>
                <option value="delay">delay</option>
                <option value="sequential">sequential</option>
              </select>
            </div>
            <div class="col-2" style="padding-right: 1rem;">
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
            <div class="col-2" style="padding-right: 1rem;">
              <input type="text" class="form-control" id="formTopicName" name="formTopicName"  v-model="form.topicName" placeholder="Topic name">
            </div>
            <div class="col" style="padding-right: 1rem;">
              <input type="text" class="form-control" id="formId" name="formId"  v-model="form.id" placeholder="Search by MessageId">
            </div>
            <div class="col-auto">
              <button type="submit" class="btn btn-primary" @click="search">{{$t('search')}}</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps,defineEmits} from "vue";

const props = defineProps({
  form:{
    id:"",
    moodType:"",
    status:"",
    topicName:""
  }
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