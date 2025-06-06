<template>
  <div class="row">
    <div class="col-12">
      <div class="row">
        <div class="col">
          <div class="form-row mb-3">
            <div class="col-2" style="padding-right: 1rem;">
              <input type="text" class="form-control" id="formChannelName" name="formChannelName"  v-model="form.channelName" placeholder="Channel name">
            </div>
            <div class="col-2" style="padding-right: 1rem;">
              <input type="text" class="form-control" id="formTopicName" name="formTopicName"  v-model="form.topicName" placeholder="Topic name">
            </div>
            <div class="col" style="padding-right: 1rem;">
              <input type="text" class="form-control" id="formId" name="formId"  v-model="form.orderKey" placeholder="Search by OrderKey">
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
    orderKey:"",
    channelName:"",
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