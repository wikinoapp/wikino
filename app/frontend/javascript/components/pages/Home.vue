<template>
  <transition name="app" mode="out-in">
    <div v-if="state.work" key="content">
    </div>
    <div v-else key="loading">
      <div class="d-flex justify-content-center align-items-center vh-100">
        <div class="c-loading">
          <div class="c-loading__core">
            Loading...
          </div>
        </div>
      </div>
    </div>
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
