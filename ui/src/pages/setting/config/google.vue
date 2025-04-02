<template>
  <div class="tab-pane fade show active" >
    <div class="container">
      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="clientId" class="col-form-label">ClientID:</label>
        </div>
        <div class="col-6">
          <input type="text" id="clientId" class="form-control" v-model="google.clientId"
                 aria-describedby="passwordHelpInline"  required>
        </div>
        <div class="col-3">
            <span id="clientIdHelpInline" class="form-text">
              create <a :href="credentials" target="_blank">credentials</a>
            </span>
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="clientSecret" class="col-form-label">ClientSecret:</label>
        </div>
        <div class="col-6">
          <input type="text" id="clientSecret" class="form-control" v-model="google.clientSecret" aria-describedby="passwordHelpInline" required>
        </div>
        <div class="col-3">
            <span id="clientSecretHelpInline" class="form-text">

            </span>
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="callBackUrl" class="col-form-label">CallBackURL:</label>
        </div>
        <div class="col-6">

          <div class="input-group mb-3">
            <button class="btn btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">{{google.scheme || "Https"}}</button>
            <ul class="dropdown-menu">
              <li><a class="dropdown-item" :class="google.scheme === 'Https' ? 'active':'' " href="javascript:;" @click="scheme('Https')">Https</a></li>
              <li><a class="dropdown-item" :class="google.scheme === 'Http' ? 'active':'' " href="javascript:;" @click="scheme('Http')">Http</a></li>
            </ul>
            <input type="url" id="callBackUrl" class="form-control" v-model="google.callBackUrl" aria-label="Text input with dropdown button" required>
          </div>

        </div>
        <div class="col-3">
              <span id="callBackUrlHelpInline" class="form-text">

              </span>
        </div>
      </div>
    </div>

  </div>
</template>
<script setup>
import { ref,defineProps,computed,defineEmits } from "vue";
const props = defineProps(["modelValue"]);
const emit = defineEmits(['update:modelValue']);

const google = computed({
  get() {
    return props['modelValue'];
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

const credentials = ref("https://console.cloud.google.com/apis/credentials?pli=1&inv=1&invt=Abs9TA");

const scheme = (val)=>{
  google.value.scheme = val;
}

</script>