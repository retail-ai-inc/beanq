<template>
  <div class="card">
    <div class="card-header bg-secondary text-white">
      <h5>Memory</h5>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Total System Memory</label>
      <div class="col-sm-5">
        {{ memory.total_system_memory_human }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Used Memory Rss</label>
      <div class="col-sm-5">
        {{ memory.used_memory_rss_human }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Used Memory Peak</label>
      <div class="col-sm-5">
        {{ memory.used_memory_peak_human }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Used Memory</label>
      <div class="col-sm-5">
        {{ memory.used_memory_human }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Used Memory Lua</label>
      <div class="col-sm-5">
        {{ memory.used_memory_lua_human }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Maxmemory</label>
      <div class="col-sm-5">
        {{ memory.maxmemory_human }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Maxmemory Policy</label>
      <div class="col-sm-5">
        {{ memory.maxmemory_policy }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Memory Rragmentation Ratio</label>
      <div class="col-sm-5">
        {{ memory.mem_fragmentation_ratio }}
      </div>
    </div>
    <div class="mb-3 row text-start p-015">
      <label class="col-sm-7 col-form-label">Memory Allocator</label>
      <div class="col-sm-5">
        {{ memory.mem_allocator }}
      </div>
    </div>

  </div>
</template>
<script setup>
import {defineProps} from "vue";

const props = defineProps({
  memory:{}
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