<template>

  <div class="tab-pane fade" id="smtp-pan" role="tabpanel" aria-labelledby="smtp-tab" tabindex="0">

  <div class="container">

    <select class="form-select"
            size="3"
            aria-label="size 3 select"
            style="border: none;overflow: hidden;margin-top: 20px;"
            v-model="email.used"
            @change="chooseSelect"
    >
      <option selected style="padding: 10px;font-weight: bold;">Choose an email service provider</option>
      <option value="smtp" style="padding: 10px">SMTP</option>
      <option value="sendGrid" style="padding: 10px">SendGrid</option>
    </select>
    <div class="smtp" v-if="email.used == 'smtp'">
    <div class="row g-3 align-items-center m-2">
      <div class="col-1 text-end">
        <label for="clientId" class="col-form-label">Host:</label>
      </div>
      <div class="col-6">
        <input type="text" id="host" class="form-control" v-model="email.smtp.host" aria-describedby="passwordHelpInline" required>
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
        <input type="text" id="port" class="form-control" v-model="email.smtp.port" aria-describedby="passwordHelpInline" required>
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
        <input type="text" id="user" class="form-control" v-model="email.smtp.user" aria-describedby="passwordHelpInline" required>
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
        <input type="text" id="password" class="form-control" v-model="email.smtp.password" aria-describedby="passwordHelpInline" required>
      </div>
      <div class="col-3">
                    <span id="callBackUrlHelpInline" class="form-text">

                    </span>
      </div>
    </div>
    </div>
    <div class="sendGrid" v-if="email.used == 'sendGrid'">
      <div class="row g-3 align-items-center m-2">
        <div class="col-1 text-end">
          <label for="clientId" class="col-form-label">Key:</label>
        </div>
        <div class="col-6">
          <input type="text" id="key" class="form-control" v-model="email.sendGrid.key" aria-describedby="passwordHelpInline" required>
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
          <input type="text" id="fromName" class="form-control" v-model="email.sendGrid.fromName" aria-describedby="passwordHelpInline" required>
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
          <input type="text" id="fromAddress" class="form-control" v-model="email.sendGrid.fromAddress" aria-describedby="passwordHelpInline" required>
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
import { defineProps,computed,defineEmits } from "vue";
const props = defineProps(["modelValue"]);
const emit = defineEmits(['update:modelValue']);

const email = computed({
  get() {
    return props['modelValue'];
  },
  set(newValue) {
    emit('update:modelValue', newValue);
  },
});

</script>