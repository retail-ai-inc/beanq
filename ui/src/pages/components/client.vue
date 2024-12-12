<template>
  <div class="card">
    <div class="card-header bg-secondary text-white">
      <h5>Client</h5>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Connected Clients</label>
      <div class="col-sm-5">
          {{ clients.connected_clients }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Blocked Clients</label>
      <div class="col-sm-5">
        {{ clients.blocked_clients }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Client Recent Max Input Buffer</label>
      <div class="col-sm-5">
        {{ clients.client_recent_max_input_buffer}}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Client Recent Max Output Buffer</label>
      <div class="col-sm-5">
        {{ clients.client_recent_max_output_buffer}}
      </div>
    </div>
  </div>
</template>
<script setup>
import {defineProps} from "vue";

const props = defineProps({
  clients: {}
})

</script>
<style>
.p-015 {
  padding: 0 1.5rem;
}
.p-015 label {
  font-weight: bold;
}
</style>