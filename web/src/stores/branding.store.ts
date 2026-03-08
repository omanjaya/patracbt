import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api/client'

export interface BrandingSettings {
  app_name: string
  app_logo: string
  app_favicon: string
  app_footer_text: string
  app_primary_color: string
  app_header_bg: string
  login_bg_image: string
  login_subtitle: string
  school_name: string
}

export const useBrandingStore = defineStore('branding', () => {
  const settings = ref<BrandingSettings>({
    app_name: 'CBT Patra',
    app_logo: '',
    app_favicon: '',
    app_footer_text: '',
    app_primary_color: '',
    app_header_bg: '',
    login_bg_image: '',
    login_subtitle: 'Masuk ke akun Anda untuk melanjutkan',
    school_name: '',
  })
  const loaded = ref(false)

  const appName = computed(() => settings.value.app_name || 'CBT Patra')
  const footerText = computed(() => settings.value.app_footer_text || settings.value.app_name || 'CBT Patra')
  const loginSubtitle = computed(() => settings.value.login_subtitle || 'Masuk ke akun Anda untuk melanjutkan')

  async function fetch() {
    try {
      const res = await api.get('/branding')
      if (res.data?.data) {
        Object.assign(settings.value, res.data.data)
      }
      // Apply favicon
      if (settings.value.app_favicon) {
        const link = document.querySelector("link[rel~='icon']") as HTMLLinkElement
        if (link) link.href = settings.value.app_favicon
      }
      // Apply primary color override
      if (settings.value.app_primary_color) {
        document.documentElement.style.setProperty('--tblr-primary', settings.value.app_primary_color)
      }
    } catch {
      // Use defaults silently
    } finally {
      loaded.value = true
    }
  }

  function refresh() {
    return fetch()
  }

  return { settings, loaded, appName, footerText, loginSubtitle, fetch, refresh }
})
