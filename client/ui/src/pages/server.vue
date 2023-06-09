<template>
    <div>
        <span class="example">{{ msg }}</span>
    </div>
</template>
  
  
<script>

export default {
    components: {
    },
    data() {
        return {
            msg: 'server!',
            color: 'blue',
        }
    }
}
</script>
  
<style scoped>
.example {
    color: v-bind('color');
}
</style>
  
  