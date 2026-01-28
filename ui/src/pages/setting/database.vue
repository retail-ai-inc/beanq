<template>
  <div class="database container-fluid">
    <table class="table table-striped">
      <thead>
      <tr>
        <th scope="col">#</th>
        <th scope="col">Collection name</th>
        <th scope="col">Documents</th>
        <th scope="col">Storage size</th>
        <th scope="col">Indexes</th>
        <th scope="col">Index size</th>
        <th scope="col">Shard</th>
      </tr>
      </thead>
      <tbody v-html="htmlR">
      </tbody>
    </table>
  </div>
</template>
<script setup>

import { ref,onMounted,onUnmounted } from "vue";

const list = ref([]);

const getDetails = async () => {
  try{
    let res = await mongoApi.List();
    list.value = res;
  }catch (e) {

  }
}

const htmlR = ref('');
function Html(data){
  let html = '';
  data.forEach((item, key) => {
    html += `<tr key="${key}" class="py-5 align-middle">
        <th>${key+1}</th>
        <td>${item.name}</td>
        <td>${item.count}</td>
        <td>${item.storageSize}</td>
        <td>${item.indexes}</td>
        <td>${item.totalIndexSize}</td>`;
    if(item.sharded){
      html += `<td><svg  xmlns="http://www.w3.org/2000/svg" width="18" height="18" fill="green" class="bi bi-check" viewBox="0 0 16 16">
            <path d="M10.97 4.97a.75.75 0 0 1 1.07 1.05l-3.99 4.99a.75.75 0 0 1-1.08.02L4.324 8.384a.75.75 0 1 1 1.06-1.06l2.094 2.093 3.473-4.425z"/>
          </svg></td>`;
    }
    if(!item.sharded){
      html += `<td><svg  xmlns="http://www.w3.org/2000/svg" width="18" height="18" fill="red" class="bi bi-x" viewBox="0 0 16 16">
            <path d="M4.646 4.646a.5.5 0 0 1 .708 0L8 7.293l2.646-2.647a.5.5 0 0 1 .708.708L8.707 8l2.647 2.646a.5.5 0 0 1-.708.708L8 8.707l-2.646 2.647a.5.5 0 0 1-.708-.708L7.293 8 4.646 5.354a.5.5 0 0 1 0-.708"/>
          </svg></td>`
    }
    html += `</tr>`;

    let nhtml = '';
    item.indexSizes.forEach((it,ik) => {
      nhtml += `<tr>
        <td>${it.Key}</td>
        <td>${it.Value}</td>
      </tr>`;
    })
    html += `<tr>
        <td></td>
        <td></td>
        <td></td>
        <td></td>
        <td></td>
        <td colspan="2">
          <table class="table">
            <thead class="table-light">
            <tr>
              <td style="font-weight: bold">Key</td>
              <td style="font-weight: bold;">Size</td>
            </tr>
            </thead>
            <tbody>
            `+nhtml+`
            </tbody>
          </table>
        </td>
      </tr>`;
  })
  return html;
}

onMounted(async () => {
  await getDetails();
  htmlR.value = Html(list.value);
})
</script>