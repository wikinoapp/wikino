<template>
  <transition name="app" mode="out-in">
    <div class="container">
      <div>
        <input class="form-control" placeholder="hello@example.com" type="email" v-model="email">
        <button type="submit" class="btn btn-primary">
          {{ $root.$t('verb.start') }}
        </button>
      </div>
      <div>
        or
      </div>
      <div>
        <a href="/auth/google_oauth2">
          {{ $root.$t('messages.signUpWithGoogle') }}
        </a>
      </div>
    </div>
  </transition>
</template>

<script lang="ts">
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
