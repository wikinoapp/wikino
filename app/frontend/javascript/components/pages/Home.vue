<template>
  <transition name="app" mode="out-in">
  </transition>
</template>

<script lang="ts">
  import $ from 'jquery'
  import { createComponent, onMounted, reactive } from '@vue/composition-api'

  export default createComponent({
    setup(props, context) {
      const state = reactive({})

      return {
        state,
      }
    }
  })
</script>
