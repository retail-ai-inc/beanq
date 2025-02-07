<template>
  <div class="role-tree">
    <ul>
      <li class="tree-item" v-for="(item,key) in nodes" :key="key">
        <input type="checkbox" :value="item.value" :pid="item.pid" @click="choose">
        <label>{{item.label}}</label>
        <div v-if="('children' in item) && item.children.length > 0">
          <tree-node :nodes="item.children"/>
        </div>
      </li>
    </ul>
  </div>
</template>
<script setup>
import {ref,defineProps,defineOptions,computed} from "vue";

defineOptions({
  name:"TreeNode"
})

const props = defineProps({
  nodes:{
    type: Object,
    required: true
  }
})

const newNodes = computed(()=>{
  props.nodes.forEach((v,k)=>{

  })
})



function choose(event){

  const pid = parseInt(event.target.attributes["pid"]) || 0;
  if(pid > 0){

  }
  console.log(event);
  console.log(event.target.attributes["pid"]);
  console.log(event.target.value);
}

</script>
<style scoped>

.role-tree{
  ul{
      list-style: none;
      li{
        display:list-item;
        padding:.25rem;
        input{
          margin:0 .25rem 0 .55rem;
        }
      }
  }
}
</style>
