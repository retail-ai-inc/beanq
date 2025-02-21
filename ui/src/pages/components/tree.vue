<template>
  <div class="role-tree">
    <ul>
      <li class="tree-item" v-for="(item,key) in nodes" :key="key">
        <input type="checkbox" :value="item.value" :pid="item.pid" :id="item.id" @click="choose"  :checked="checkedIds.includes(item.id) ? 'checked' : false">
        <label>{{item.label}}</label>
        <div v-if="('children' in item) && item.children.length > 0">
          <tree-node :nodes="item.children" :checkedIds="checkedIds" @choose="treeChoose"/>
        </div>
      </li>
    </ul>
  </div>
</template>
<script setup>
import {defineProps,defineOptions,defineEmits} from "vue";

defineOptions({
  name:"TreeNode"
})

const props = defineProps({
  nodes:{
    type: Array,
    required: true
  },
  checkedIds:{
    type: Array,
    default: [],
    required: true
  }
})

const emits = defineEmits(['choose']);
function treeChoose(event){
  emits("choose",event);
}

const choose = function (event){
  emits("choose",event);
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
