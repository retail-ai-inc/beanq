<template>
  <div>
    <GoBackButton/>

    <div class="container-fluid" style="padding: 1.5rem;border-radius: 0.25rem">
      <div class="row" v-for="(item,key) in detail" style="min-height: 2.5rem">
        <div class="col-1" style="font-weight: bold">
          {{key}}
        </div>
        <div class="col mark" style="white-space: pre-wrap;">
          <pre v-if="key === 'payload'" class="payload-pre"><code class="payload-code">{{ JSON.stringify(JSON.parse(item), null, 2)}}</code></pre>
          <pre v-else><code>{{item}}</code></pre>
        </div>
      </div>
    </div>
    <GoBackButton />
  </div>
</template>
<script setup>

import {ref,onMounted} from "vue";
import { useRoute } from 'vueRouter';
import GoBackButton from "../../components/goBackButton.vue";

function getDetail(id){
  return request.get("/event_log/detail",{"params":{"id":id}})
}

let detail = ref({});
const uRoute = useRoute();

onMounted(async ()=>{
  let paramid = uRoute.params.id;
  let res = await getDetail(paramid);
  let {_id,id,...data} = res;
  detail.value = {ObjectId:_id,"Message Id":id,...data};
})


</script>
<style scoped>
.mark{
  .payload-pre{
    background-color: #f5f5f5;
    padding: 10px;
    border-radius: 5px;
    border: 1px solid #ddd;
    overflow: auto;
  }
  .payload-code{
    color: #d63384;
  }
}
</style>