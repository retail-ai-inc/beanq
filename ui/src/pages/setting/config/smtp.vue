<template>
  <div class="tab-pane fade" id="smtp-pan" role="tabpanel" aria-labelledby="smtp-tab" tabindex="0">
  <div class="container">
    <div class="row g-3 align-items-center m-2">
      <div class="col-1 text-end">
        <label for="clientId" class="col-form-label">Host:</label>
      </div>
      <div class="col-6">
        <input type="text" id="host" class="form-control" v-model="smtp.host" aria-describedby="passwordHelpInline" required>
      </div>
      <div class="col-3">
                    <span id="clientIdHelpInline" class="form-text">

                    </span>
      </div>
    </div>

    <div class="row g-3 align-items-center m-2">
      <div class="col-1 text-end">
        <label for="clientSecret" class="col-form-label">Port:</label>
      </div>
      <div class="col-6">
        <input type="text" id="port" class="form-control" v-model="smtp.port" aria-describedby="passwordHelpInline" required>
      </div>
      <div class="col-3">
                    <span id="clientSecretHelpInline" class="form-text">

                    </span>
      </div>
    </div>

    <div class="row g-3 align-items-center m-2">
      <div class="col-1 text-end">
        <label for="callBackUrl" class="col-form-label">User:</label>
      </div>
      <div class="col-6">
        <input type="text" id="user" class="form-control" v-model="smtp.user" aria-describedby="passwordHelpInline" required>
      </div>
      <div class="col-3">
                    <span id="callBackUrlHelpInline" class="form-text">

                    </span>
      </div>
    </div>
    <div class="row g-3 align-items-center m-2">
      <div class="col-1 text-end">
        <label for="callBackUrl" class="col-form-label">Password:</label>
      </div>
      <div class="col-6">
        <input type="text" id="password" class="form-control" v-model="smtp.password" aria-describedby="passwordHelpInline" required>
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
import { defineProps,computed,defineEmits } from "vue";
const props = defineProps(["modelValue"]);
const emit = defineEmits(['update:modelValue']);

const smtp = computed({
  get() {
    return props['modelValue'];
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

</script>