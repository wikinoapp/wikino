import ujs from '@rails/ujs'
import 'bootstrap'
import Turbolinks from 'turbolinks'
import Vue from 'vue'
import VueCompositionApi from '@vue/composition-api'
import VueI18n from 'vue-i18n'

import Home from './components/pages/Home.vue'

Vue.config.productionTip = false
Vue.use(VueCompositionApi)
Vue.use(VueI18n)

const i18n = new VueI18n()

Vue.component('p-home', Home)

document.addEventListener('turbolinks:load', _event => {
  new Vue({
    i18n,
    el: '#app',
    data() {
      return {
        viewer: null,
      }
    },

    methods: {
      isSignedIn() {
        return !!this.viewer
      },
    },
  })
})

ujs.start()
Turbolinks.start()
