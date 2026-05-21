<template>
  <div class="tab-pane fade" id="google-recaptcha-pan" role="tabpanel" aria-labelledby="google-recaptcha-tab" tabindex="0">
  <div class="tab-pane fade show active" >
    <div class="container">
      <div class="row g-3 align-items-center m-2">
        <p class="lead fs-6">
          For account security, BeanQ supports Google reCAPTCHA login verification to securely access the BeanQ dashboard. <br/>
          To generate v2 and v3 ID codes for Google reCAPTCHA, you must first register the <a href="https://www.google.com/recaptcha/admin/create" target="_blank">ID code</a> and <a href="https://console.cloud.google.com/security/recaptcha/create?project=testlogin-354701" target="_blank">apiKey</a>.
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
          <label for="projectId" class="col-form-label">ProjectID:</label>
        </div>
        <div class="col-6">
          <input type="text" id="projectId" class="form-control" v-model="googleRecaptcha.projectId"
                 aria-describedby="passwordHelpInline">
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="siteKeyV2" class="col-form-label">SiteKeyV2:</label>
        </div>
        <div class="col-6">
          <input type="text" id="siteKeyV2" class="form-control" v-model="googleRecaptcha.siteKeyV2" aria-describedby="passwordHelpInline">
        </div>
        <div class="col-3">
            <span id="clientSecretHelpInline" class="form-text">

            </span>
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="siteKeyV3" class="col-form-label">SiteKeyV3:</label>
        </div>
        <div class="col-6">

          <div class="input-group mb-3">
            <input type="text" id="siteKeyV3" class="form-control" v-model="googleRecaptcha.siteKeyV3" aria-label="Text input with dropdown button">
          </div>

        </div>
        <div class="col-3">
              <span id="callBackUrlHelpInline" class="form-text">

              </span>
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="apiKey" class="col-form-label">ApiKey:</label>
        </div>
        <div class="col-6">

          <div class="input-group mb-3">
            <input type="text" id="apiKey" class="form-control" v-model="googleRecaptcha.apiKey" aria-label="Text input with dropdown button">
          </div>

        </div>
        <div class="col-3">
              <span id="callBackUrlHelpInline" class="form-text">

              </span>
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