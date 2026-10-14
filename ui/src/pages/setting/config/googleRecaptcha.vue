<template>
  <div class="tab-pane fade" id="google-recaptcha-pan" role="tabpanel" aria-labelledby="google-recaptcha-tab" tabindex="0">
  <div class="tab-pane fade show active" >
    <div class="container">
      <div class="row g-3 align-items-center m-2">
        <p class="lead fs-6">
          For account security, BeanQ uses Google Cloud Fraud Defense (reCAPTCHA Universal key) on login. Challenges are decided by your key policy; the dashboard only executes the key and the backend creates an assessment. <br/>
          Create a <a href="https://console.cloud.google.com/security/recaptcha/keys" target="_blank">Universal Site Key</a> and an <a href="https://console.cloud.google.com/apis/credentials" target="_blank">API key</a> with reCAPTCHA Enterprise permission.
        </p>
        <div class="col-1 text-end">
          <label for="isActive" class="col-form-label">Active:</label>
        </div>
        <div class="col-6">
          <div class="form-check form-switch">
            <input class="form-check-input" type="checkbox" role="switch" id="isActive" checked v-model="googleRecaptcha.active" style="width: 65px;height:15px;cursor: pointer;">
            <label class="form-check-label" for="isActive">Open to use</label>
          </div>
        </div>

      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="projectId" class="col-form-label">ProjectId:</label>
        </div>
        <div class="col-6">
          <input type="text" id="projectId" class="form-control" v-model="googleRecaptcha.projectId" aria-describedby="passwordHelpInline">
        </div>
        <div class="col-3">
            <span id="clientSecretHelpInline" class="form-text">

            </span>
        </div>
      </div>
      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="siteKey" class="col-form-label">SiteKey:</label>
        </div>
        <div class="col-6">
          <input type="text" id="siteKey" class="form-control" v-model="googleRecaptcha.siteKey"
                 aria-describedby="passwordHelpInline">
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="apiKey" class="col-form-label">ApiKey:</label>
        </div>
        <div class="col-6">
          <input type="text" id="apiKey" class="form-control" v-model="googleRecaptcha.apiKey"
                 aria-describedby="passwordHelpInline">
        </div>
      </div>
      </div>
    </div>

  </div>
  </div>
</template>
<script setup>
import { ref,defineProps,computed,defineEmits } from "vue";
const props = defineProps(["modelValue"]);
const emit = defineEmits(['update:modelValue']);

const googleRecaptcha = computed({
  get() {
    return props['modelValue'];
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

</script>