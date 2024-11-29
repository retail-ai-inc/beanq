<template>
  <div>

    <div class="container-fluid" style="background: #212529;color:#fff;padding: 1.5rem;border-radius: 0.25rem">
      <div class="row" v-for="(item,key) in detail" style="min-height: 2.5rem">
        <div class="col-1" style="font-weight: bold">
          {{key}}
        </div>
        <div class="col" style="white-space: pre-wrap;">
          {{item}}
        </div>
      </div>
    </div>

  </div>
</template>
<script setup>

import {reactive,toRefs,onMounted} from "vue";
import { useRoute,useRouter } from 'vueRouter';

function getDetail(id){
  return request.get("/event_log/detail",{"params":{"id":id}})
}

let data = reactive({
  detail:{}
});

const uRoute = useRoute();
onMounted(async ()=>{
  let id = uRoute.params.id;
  let res = await getDetail(id);
  data.detail = res.data;
})
const {detail} = toRefs(data);
</script>