<template>
  <div class="config container-fluid">
      <div class="row mb-4">
        <div class="col">
          <h5 class="card-title">Configuration Information</h5>
        </div>
      </div>
    <div>
      <span>Google</span>
      <div class="line" style="margin: 1rem 0;border-bottom: 1px solid #000;transform: scaleY(0.2);"></div>

      <div style="margin-left: 5rem">
        <div class="row g-3 align-items-center" style="margin-top: .5rem;">
          <div class="col-1 text-end">
            <label for="inputPassword6" class="col-form-label">ClientID:</label>
          </div>
          <div class="col-6">
            <input type="password" id="inputPassword6" class="form-control" aria-describedby="passwordHelpInline">
          </div>
          <div class="col-3">
          <span id="passwordHelpInline" class="form-text">
            Must be 8-20 characters long.
          </span>
          </div>
        </div>

        <div class="row g-3 align-items-center" style="margin-top: .5rem;">
          <div class="col-1 text-end">
            <label for="inputPassword6" class="col-form-label">ClientSecret:</label>
          </div>
          <div class="col-6">
            <input type="password" id="inputPassword6" class="form-control" aria-describedby="passwordHelpInline">
          </div>
          <div class="col-3">
        <span id="passwordHelpInline" class="form-text">
          Must be 8-20 characters long.
        </span>
          </div>
        </div>

        <div class="row g-3 align-items-center" style="margin-top: .5rem;">
          <div class="col-1 text-end">
            <label for="inputPassword6" class="col-form-label">CallBackURL:</label>
          </div>
          <div class="col-6">
            <input type="password" id="inputPassword6" class="form-control" aria-describedby="passwordHelpInline">
          </div>
          <div class="col-3">
        <span id="passwordHelpInline" class="form-text">
          Must be 8-20 characters long.
        </span>
          </div>
        </div>
      </div>

      <div class="line" style="margin: 1rem 0;border-bottom: 1px solid #000;transform: scaleY(0.2);"></div>
      <button type="button" class="btn btn-primary">{{$t('edit')}}</button>
    </div>
  </div>
</template>
<script setup>
</script>