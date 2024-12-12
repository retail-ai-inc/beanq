<template>
  <div class="card">
    <div class="card-header bg-secondary text-white">
      <h5>Stats</h5>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Total Connections Received</label>
      <div class="col-sm-5">
        {{ stats.total_connections_received }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Total Commands Processed</label>
      <div class="col-sm-5">
        {{ stats.total_commands_processed }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Instantaneous Ops Per Sec</label>
      <div class="col-sm-5">
        {{ stats.instantaneous_ops_per_sec }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Total Net Input Bytes</label>
      <div class="col-sm-5">
        {{ stats.total_net_input_bytes }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Total Net Output Bytes</label>
      <div class="col-sm-5">
        {{ stats.total_net_output_bytes }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Instantaneous Input Kbps</label>
      <div class="col-sm-5">
        {{ stats.instantaneous_input_kbps }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Instantaneous Input Kbps</label>
      <div class="col-sm-5">
        {{ stats.instantaneous_output_kbps }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Rejected Connections</label>
      <div class="col-sm-5">
        {{ stats.rejected_connections }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Expired Keys</label>
      <div class="col-sm-5">
        {{ stats.expired_keys }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Evicted Keys</label>
      <div class="col-sm-5">
        {{ stats.evicted_keys }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Keyspace Hits</label>
      <div class="col-sm-5">
        {{ stats.keyspace_hits }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Keyspace Misses</label>
      <div class="col-sm-5">
        {{ stats.keyspace_misses }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Latest Fork Usec</label>
      <div class="col-sm-5">
        {{ stats.latest_fork_usec }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Migrate Cached Sockets</label>
      <div class="col-sm-5">
        {{ stats.migrate_cached_sockets }}
      </div>
    </div>

  </div>
</template>
<script setup>
import {defineProps} from "vue";

const props = defineProps({
  stats:{}
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