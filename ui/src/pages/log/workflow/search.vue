<template>
  <div class="row">
      <div class="col-2" style="padding-right: 1rem;">
        <div class="form-row">
          <div class="col">
            <select class="form-select" aria-label="Default select" id="formStatus" name="formStatus" style="cursor: pointer" v-model="form.status">
              <option selected value="">All status</option>
              <option value="prepared">prepared</option>
              <option value="succeed">succeed</option>
              <option value="failed">failed</option>
              <option value="aborting">aborting</option>
            </select>
          </div>
        </div>
      </div>
      <div class="col-2" style="padding-right: 1rem;">
        <input type="text" class="form-control" id="formChannelName" name="formChannelName"  v-model="form.channelName" placeholder="Channel Name">
      </div>
      <div class="col-2" style="padding-right: 1rem;">
        <input type="text" class="form-control" id="formTopicName" name="formTopicName"  v-model="form.topicName" placeholder="Topic name">
      </div>
      <div class="col-auto">
        <button type="submit" class="btn btn-primary" @click="search">{{$t('search')}}</button>
      </div>
  </div>
</template>
<script setup>
import {defineProps,defineEmits} from "vue";

const props = defineProps({
  form:{
    channelName:"",
    topicName:"",
    status:""
  }
})
const emits = defineEmits(['search']);
const search = function (){
  emits("search");
}
</script>