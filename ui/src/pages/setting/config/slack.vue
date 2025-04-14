<template>
  <div class="tab-pane fade" id="smtp-pan" role="tabpanel" aria-labelledby="smtp-tab" tabindex="0">
  <div class="container">
    <div class="row g-3 align-items-center m-2">
      <div class="col-2 text-end">
        <label for="clientId" class="col-form-label">Bot Auth Token:</label>
      </div>
      <div class="col-6">
        <input type="text" id="slack" class="form-control" v-model="slack.botAuthToken" aria-describedby="passwordHelpInline" required>
      </div>
      <div class="col-3">
        <span id="clientIdHelpInline" class="form-text">

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

const slack = computed({
  get() {
    return props['modelValue'];
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

</script>