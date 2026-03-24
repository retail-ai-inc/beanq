<template>
  <div class="tab-pane fade show active" >
    <div class="container">
      <div class="row g-3 align-items-center m-2">
        <p class="lead">
          If you use Google Workspace, you can integrate your Google account in BeanQ to access the BeanQ Dashboard. To generate a <strong>Client ID</strong> and <strong>Client Secret</strong> for Google Login (OAuth 2.0), you must use the <a href="https://console.cloud.google.com/" target="_blank">Google Cloud Console</a>. These credentials identify your application to Google's authentication servers.
        </p>
        <div class="col-1 text-end">
          <label for="clientId" class="col-form-label">ClientID:</label>
        </div>
        <div class="col-6">
          <input type="text" id="clientId" class="form-control" v-model="google.clientId"
                 aria-describedby="passwordHelpInline">
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="clientSecret" class="col-form-label">ClientSecret:</label>
        </div>
        <div class="col-6">
          <input type="text" id="clientSecret" class="form-control" v-model="google.clientSecret" aria-describedby="passwordHelpInline">
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
            <input type="url" id="callBackUrl" disabled readonly class="form-control" v-model="google.callBackUrl" aria-label="Text input with dropdown button">
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

</script>