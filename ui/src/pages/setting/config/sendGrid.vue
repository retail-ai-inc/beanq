<template>
  <div class="tab-pane fade" id="send-grid-pane" role="tabpanel" aria-labelledby="send-grid-tab" tabindex="0">
    <div class="container">
      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="clientId" class="col-form-label">Key:</label>
        </div>
        <div class="col-6">
          <input type="text" id="key" class="form-control" v-model="grid.key" aria-describedby="passwordHelpInline" required>
        </div>
        <div class="col-3">
                    <span id="clientIdHelpInline" class="form-text">

                    </span>
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="clientSecret" class="col-form-label">FromName:</label>
        </div>
        <div class="col-6">
          <input type="text" id="fromName" class="form-control" v-model="grid.fromName" aria-describedby="passwordHelpInline" required>
        </div>
        <div class="col-3">
                    <span id="clientSecretHelpInline" class="form-text">

                    </span>
        </div>
      </div>

      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="callBackUrl" class="col-form-label">FromAddress:</label>
        </div>
        <div class="col-6">
          <input type="text" id="fromAddress" class="form-control" v-model="grid.fromAddress" aria-describedby="passwordHelpInline" required>
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

const grid = computed({
  get() {
    return props['modelValue'];
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

</script>