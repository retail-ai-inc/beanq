<template>
  <div class="container-fluid">
    <div class="row mb-4">
      <div class="col">
        <h5 class="card-title">List of DB Keys</h5>
      </div>
    </div>
    <Spinner v-if="loading"/>
    <div v-else class="db-size">
      <NoMessage v-if="dbs.length <= 0"/>
      <table v-else class="table table-striped table-hover">
        <thead>
        <tr>
          <th scope="col">#</th>
          <th scope="col">Key</th>
          <th scope="col">Action</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="(item, index) in dbs" :key="index">
          <td>{{index+1}}</td>
          <td>{{item}}</td>
          <td>
            <Delete_icon @action="deleteModalItem(item)"/>
          </td>
        </tr>
        </tbody>
      </table>
    </div>

    <Action :label="deleteLabel" :id="showDeleteModal" :data-id="deleteId" :warning="$t('retryWarningHtml')" :info="$t('retryInfoHtml')" @action="deleteInfo">
      <template #title="{title}">
        {{$t("sureDelete")}}
      </template>
    </Action>
    <Btoast :id="toastId" ref="toastRef"/>
    <LoginModal :id="loginId" ref="loginModal"/>
  </div>
</template>
<script setup>
import { ref, onMounted } from 'vue';
import Delete_icon from "../components/icons/delete_icon.vue";
import Action from "../components/action.vue";
import Spinner from "../components/spinner.vue";
import LoginModal from "../components/loginModal.vue";
import Btoast from "../components/btoast.vue";
import NoMessage from "../components/noMessage.vue";

const [dbs] = [ref([])];
const loading = ref(false);
const [
    deleteLabel,
  showDeleteModal,
  deleteId,
  modal] = [
      ref("deleteLabel"),
  ref("showDeleteModal"),
  ref(""),
  ref(null)];
const [loginId,loginModal] = [ref("staticBackdrop"),ref("loginModal")];
const [toastRef,toastId] = [ref(null),ref("toastId")]

const getKeys = async () => {
  loading.value = true;
  try {
    await request.get('/redis/keys').then(res => {
      dbs.value = res;
      setTimeout(()=>{
        loading.value = false;
      },500);
    });
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
}

const deleteModalItem= async (item)=>{

  deleteId.value = "";
  const eleRetry = document.getElementById("showDeleteModal");
  modal.value = new bootstrap.Modal(eleRetry);
  modal.value.show(eleRetry);
  deleteId.value = item;
}

const deleteInfo = async ()=>{
  try {
    await request.delete(`/redis/${deleteId.value}`,{
      data:{
        key:deleteId.value
      }
    }).then(res=>{
      toastRef.value.show("success");
      modal.value.hide();
      getKeys();
    });
  }catch (err) {
    //401 error
    if (err?.response?.status === 401){
      loginModal.value.error(err);
      return;
    }
    //normal error
    toastRef.value.show(err);
  }
}

onMounted(()=>{
  getKeys();
})
</script>