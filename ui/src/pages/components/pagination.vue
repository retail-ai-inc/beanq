<template>
  <nav aria-label="Page navigation">
    <ul class="pagination justify-content-end" v-if="page <= total">
      <li class="page-item" :class="page === 1 ? 'disabled' : ''" >
        <a class="page-link" @click="changePage(page-1 <= 0 ? 1 : page-1,cursor)">&laquo;</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="page !== 1" @click="changePage(1,cursor)">1</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="(page -2) > 1">...</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="(page - 2) > 1" @click="changePage(page-2,cursor)">{{page -2}}</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="(page - 1) > 1" @click="changePage(page-1,cursor)">{{page -1}}</a>
      </li>
      <li class="page-item active" aria-current="page">
        <a class="page-link">{{page}}</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="(page + 1) < total" @click="changePage(page+1,cursor)">{{page + 1}}</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="(page + 2) < total" @click="changePage(page+2,cursor)">{{page + 2}}</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="(page + 2) < total">...</a>
      </li>
      <li class="page-item">
        <a class="page-link" v-if="page != total && total > 0" @click="changePage(total,cursor)">{{total}}</a>
      </li>
      <li class="page-item" :class="page == total ? 'disabled' : ''" >
        <a class="page-link" @click="changePage((page+1) >= total ? total : (page + 1),cursor)">&raquo;</a>
      </li>
    </ul>
  </nav>
</template>
<script setup>
import {defineProps,defineEmits} from "vue";

const props = defineProps({
  page:1,
  total:1,
  cursor:0
})

const emits = defineEmits(['changePage']);
const changePage = function (page,cursor){
  emits("changePage",page,cursor);
}
</script>
<style scoped>
.pagination .page-link{
  cursor: pointer;
}
</style>