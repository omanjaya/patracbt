<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { settingApi } from '../../../api/setting.api'
import { useToastStore } from '../../../stores/toast.store'
import { useBrandingStore } from '@/stores/branding.store'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const router = useRouter()
const branding = useBrandingStore()

const toast = useToastStore()

const activeTab = ref('umum')

const loading = ref(true)
const saving = ref(false)
const backingUp = ref(false)
const restoring = ref(false)
const testingAI = ref(false)
const clearingCache = ref(false)
const restoreFileInput = ref<HTMLInputElement | null>(null)
const uploadingBranding = ref(false)

// Panic Mode state
const panicModeActive = ref(false)
const panicModeLoading = ref(false)
const showPanicConfirm = ref(false)
const showClearCacheConfirm = ref(false)

// Form validation
const formErrors = reactive({ app_name: '' })

// MinIO state
const savingMinio = ref(false)
const testingMinio = ref(false)
const backingUpMinio = ref(false)
const restoringMinio = ref(false)
const minioBackups = ref<{ key: string; last_modified: string; size: number }[]>([])
const loadingMinioList = ref(false)
const selectedMinioFile = ref('')
const showMinioBackupList = ref(false)

const form = reactive<Record<string, string>>({
  ip_whitelist_enabled: '0',
  ip_whitelist_ips: '',
  app_name: '',
  ai_api_url: '',
  ai_api_key: '',
  ai_api_header: 'Authorization',
  ai_model_params: '{}',
  websocket_enabled: '1',
  panic_mode_active: '0',
  login_method: 'normal',
  app_logo: '',
  app_favicon: '',
  app_footer_text: '',
  app_primary_color: '#206bc4',
  app_header_bg: '',
  login_bg_image: '',
  login_subtitle: '',
  school_name: '',
  pwa_app_name: '',
  pwa_short_name: '',
  pwa_theme_color: '#ffffff',
  pwa_background_color: '#ffffff',
})

const minioForm = reactive({
  minio_endpoint: '',
  minio_bucket: '',
  minio_access_key: '',
  minio_secret_key: '',
  minio_use_ssl: '0',
})

// IP Whitelist state
const ipWhitelistEnabled = ref(false)
const ipWhitelistIPs = ref('')
const savingWhitelist = ref(false)
const myIP = ref('')
const loadingMyIP = ref(false)

const sidebarTabs = [
  { key: 'umum', label: 'Umum & Branding', icon: 'ti-settings' },
  { key: 'ip-whitelist', label: 'IP Whitelist', icon: 'ti-shield-lock' },
  { key: 'ai', label: 'Konfigurasi AI', icon: 'ti-robot' },
  { key: 'backup', label: 'Backup & Restore', icon: 'ti-database' },
  { key: 'database', label: 'Manajemen Database', icon: 'ti-database-cog' },
  { key: 'minio', label: 'MinIO Cloud', icon: 'ti-cloud' },
  { key: 'pwa', label: 'PWA', icon: 'ti-device-mobile' },
  { key: 'panic', label: 'Panic Mode', icon: 'ti-alert-triangle', badge: true },
  { key: 'cache', label: 'Cache & Performa', icon: 'ti-bolt' },
]

async function fetchSettings() {
  loading.value = true
  try {
    const res = await settingApi.getAll()
    Object.assign(form, res.data.data)
    // Sync IP whitelist state
    ipWhitelistEnabled.value = res.data.data?.ip_whitelist_enabled === '1'
    // Convert comma-separated to newline-separated for textarea display
    const rawIPs = res.data.data?.ip_whitelist_ips || ''
    ipWhitelistIPs.value = rawIPs.split(',').map((s: string) => s.trim()).filter(Boolean).join('\n')

    // Sync minio form from settings
    minioForm.minio_endpoint = res.data.data?.minio_endpoint || ''
    minioForm.minio_bucket = res.data.data?.minio_bucket || ''
    minioForm.minio_access_key = res.data.data?.minio_access_key || ''
    minioForm.minio_secret_key = res.data.data?.minio_secret_key || ''
    minioForm.minio_use_ssl = res.data.data?.minio_use_ssl || '0'
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat pengaturan')
  } finally {
    loading.value = false
  }
}

async function fetchPanicModeStatus() {
  try {
    const res = await settingApi.panicModeStatus()
    panicModeActive.value = res.data.data?.active === true
  } catch {
    // ignore — fallback to form value
    panicModeActive.value = form.panic_mode_active === '1'
  }
}

function validateSettings(): boolean {
  formErrors.app_name = ''
  if (!form.app_name || form.app_name.trim().length < 1) {
    formErrors.app_name = 'Nama aplikasi wajib diisi'
    return false
  }
  return true
}

async function handleSave() {
  if (!validateSettings()) {
    toast.error('Periksa kembali isian form')
    return
  }
  saving.value = true
  try {
    await settingApi.update({ ...form })
    toast.success('Pengaturan berhasil disimpan')
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menyimpan pengaturan')
  } finally {
    saving.value = false
  }
}

async function handleBackup() {
  backingUp.value = true
  try {
    const res = await settingApi.createBackup()
    const filename = res.data.data?.filename
    if (filename) {
      const url = settingApi.downloadBackupUrl(filename)
      const a = document.createElement('a')
      a.href = url; a.download = filename; a.click()
      toast.success('Backup berhasil diunduh')
    }
  } catch {
    toast.error('Gagal membuat backup')
  } finally {
    backingUp.value = false
  }
}

async function handleRestoreFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  restoring.value = true
  try {
    await settingApi.restoreBackup(file)
    toast.success('Restore berhasil — pengaturan telah dipulihkan')
    await fetchSettings()
  } catch {
    toast.error('Gagal melakukan restore')
  } finally {
    restoring.value = false
    if (restoreFileInput.value) restoreFileInput.value.value = ''
  }
}

async function handleTestAI() {
  testingAI.value = true
  try {
    const res = await settingApi.testAI()
    toast.success(res.data.data?.message || 'Koneksi AI berhasil')
  } catch (err: any) {
    toast.error(err?.response?.data?.message || 'Koneksi AI gagal')
  } finally {
    testingAI.value = false
  }
}

function askClearCache() {
  showClearCacheConfirm.value = true
}

async function handleClearCache() {
  showClearCacheConfirm.value = false
  clearingCache.value = true
  try {
    const res = await settingApi.clearCache()
    toast.success(res.data.data?.message || 'Cache berhasil dibersihkan')
  } catch (err: any) {
    toast.error(err?.response?.data?.message || 'Gagal membersihkan cache')
  } finally {
    clearingCache.value = false
  }
}

// Panic Mode handlers
function askTogglePanicMode() {
  showPanicConfirm.value = true
}

async function handleTogglePanicMode() {
  showPanicConfirm.value = false
  panicModeLoading.value = true
  try {
    if (panicModeActive.value) {
      await settingApi.deactivatePanicMode()
      panicModeActive.value = false
      form.panic_mode_active = '0'
      toast.success('Panic Mode berhasil dinonaktifkan. Semua peserta telah menerima notifikasi dan dapat melanjutkan ujian.')
    } else {
      await settingApi.activatePanicMode()
      panicModeActive.value = true
      form.panic_mode_active = '1'
      toast.success('Panic Mode berhasil diaktifkan! Semua peserta telah menerima notifikasi dan akses ujian dikunci secara instan.')
    }
  } catch {
    toast.error('Gagal mengubah status Panic Mode')
  } finally {
    panicModeLoading.value = false
  }
}

// MinIO handlers
async function handleSaveMinio() {
  savingMinio.value = true
  try {
    await settingApi.saveMinioSettings({ ...minioForm })
    toast.success('Konfigurasi MinIO berhasil disimpan')
  } catch {
    toast.error('Gagal menyimpan konfigurasi MinIO')
  } finally {
    savingMinio.value = false
  }
}

async function handleTestMinio() {
  testingMinio.value = true
  try {
    const res = await settingApi.testMinioConnection()
    const data = res.data.data
    if (data?.success) {
      toast.success(data.message || 'Koneksi MinIO berhasil')
    } else {
      toast.error(data?.message || 'Koneksi MinIO gagal')
    }
  } catch (err: any) {
    toast.error(err?.response?.data?.message || 'Koneksi MinIO gagal')
  } finally {
    testingMinio.value = false
  }
}

async function handleBackupToMinio() {
  backingUpMinio.value = true
  try {
    const res = await settingApi.backupToMinio()
    toast.success(res.data.data?.message || 'Backup berhasil diupload ke MinIO')
  } catch (err: any) {
    toast.error(err?.response?.data?.message || 'Gagal backup ke MinIO')
  } finally {
    backingUpMinio.value = false
  }
}

async function handleListMinioBackups() {
  loadingMinioList.value = true
  showMinioBackupList.value = true
  try {
    const res = await settingApi.listMinioBackups()
    minioBackups.value = res.data.data || []
    if (minioBackups.value.length === 0) {
      toast.success('Tidak ada file backup di MinIO')
    }
  } catch (err: any) {
    toast.error(err?.response?.data?.message || 'Gagal mengambil daftar backup MinIO')
    showMinioBackupList.value = false
  } finally {
    loadingMinioList.value = false
  }
}

async function handleRestoreFromMinio() {
  if (!selectedMinioFile.value) {
    toast.error('Pilih file backup terlebih dahulu')
    return
  }
  restoringMinio.value = true
  try {
    const res = await settingApi.restoreFromMinio(selectedMinioFile.value)
    toast.success(res.data.data?.message || 'Pengaturan berhasil dipulihkan dari MinIO')
    await fetchSettings()
    showMinioBackupList.value = false
    selectedMinioFile.value = ''
  } catch (err: any) {
    toast.error(err?.response?.data?.message || 'Gagal restore dari MinIO')
  } finally {
    restoringMinio.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatDate(iso: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('id-ID')
}

async function handleBrandingUpload(event: Event, field: string) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploadingBranding.value = true
  try {
    const res = await settingApi.uploadBranding(field, file)
    if (res.data.data?.url) {
      form[field] = res.data.data.url
      toast.success('File berhasil diupload')
      const brandingStore = useBrandingStore()
      brandingStore.refresh()
    }
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal mengupload file')
  } finally {
    uploadingBranding.value = false;
    (event.target as HTMLInputElement).value = ''
  }
}

async function handleSaveWhitelist() {
  savingWhitelist.value = true
  try {
    // Convert newline-separated to comma-separated for storage
    const ips = ipWhitelistIPs.value.split('\n').map(s => s.trim()).filter(Boolean).join(',')
    await settingApi.update({
      ip_whitelist_enabled: ipWhitelistEnabled.value ? '1' : '0',
      ip_whitelist_ips: ips,
    })
    toast.success('IP Whitelist berhasil disimpan')
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menyimpan IP Whitelist')
  } finally {
    savingWhitelist.value = false
  }
}

async function detectMyIP() {
  loadingMyIP.value = true
  try {
    const res = await fetch('https://api.ipify.org?format=json')
    const data = await res.json()
    myIP.value = data.ip
  } catch {
    toast.error('Gagal mendeteksi IP')
  } finally {
    loadingMyIP.value = false
  }
}

function addMyIP() {
  if (!myIP.value) return
  const current = ipWhitelistIPs.value.split('\n').map(s => s.trim()).filter(Boolean)
  if (!current.includes(myIP.value)) {
    current.push(myIP.value)
    ipWhitelistIPs.value = current.join('\n')
  }
}

onMounted(async () => {
  await fetchSettings()
  await fetchPanicModeStatus()
})
</script>
<template>
  <BasePageHeader
    title="Pengaturan"
    :subtitle="`Konfigurasi aplikasi ${branding.appName}`"
    :breadcrumbs="[{ label: 'Pengaturan' }]"
  >
    <template #actions>
      <BaseButton variant="primary" :loading="saving" @click="handleSave">
        <i class="ti ti-device-floppy me-1"></i>Simpan Pengaturan
      </BaseButton>
    </template>
  </BasePageHeader>

  <div v-if="loading" class="row g-3">
    <div class="col-lg-3">
      <div class="card">
        <div class="card-body">
          <div v-for="n in 6" :key="n" class="placeholder-glow mb-3">
            <span class="placeholder col-12" style="height: 32px; border-radius: 4px;"></span>
          </div>
        </div>
      </div>
    </div>
    <div class="col-lg-9">
      <div class="card">
        <div class="card-header">
          <div class="placeholder-glow" style="width: 40%;">
            <span class="placeholder col-12"></span>
          </div>
        </div>
        <div class="card-body d-flex flex-column gap-3">
          <div v-for="m in 4" :key="m">
            <div class="placeholder-glow mb-1" style="width: 30%;">
              <span class="placeholder placeholder-xs col-12"></span>
            </div>
            <div class="placeholder-glow">
              <span class="placeholder col-12" style="height: 36px; border-radius: 4px;"></span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="row g-3">
    <!-- Sidebar Navigation - Desktop -->
    <div class="col-lg-3 d-none d-lg-block">
      <div class="card" style="position: sticky; top: 4rem;">
        <div class="list-group list-group-flush">
          <a
            v-for="tab in sidebarTabs"
            :key="tab.key"
            href="javascript:void(0)"
            class="list-group-item list-group-item-action d-flex align-items-center"
            :class="{ 'active': activeTab === tab.key }"
            @click="activeTab = tab.key"
          >
            <i class="ti me-2" :class="[tab.icon, tab.key === 'panic' && panicModeActive ? 'text-danger' : '']"></i>
            <span class="flex-fill">{{ tab.label }}</span>
            <span v-if="tab.badge && panicModeActive" class="badge bg-danger badge-pill ms-2">!</span>
          </a>
        </div>
      </div>
    </div>

    <!-- Sidebar Navigation - Mobile (horizontal scrollable) -->
    <div class="col-12 d-lg-none">
      <div class="card">
        <div class="card-body p-2">
          <div class="settings-mobile-nav">
            <a
              v-for="tab in sidebarTabs"
              :key="tab.key"
              href="javascript:void(0)"
              class="settings-mobile-nav-item"
              :class="{ 'active': activeTab === tab.key }"
              @click="activeTab = tab.key"
            >
              <i class="ti" :class="[tab.icon, tab.key === 'panic' && panicModeActive ? 'text-danger' : '']"></i>
              <span>{{ tab.label }}</span>
              <span v-if="tab.badge && panicModeActive" class="badge bg-danger badge-pill ms-1" style="font-size: 0.6rem;">!</span>
            </a>
          </div>
        </div>
      </div>
    </div>

    <!-- Content Area -->
    <div class="col-lg-9">

      <!-- Tab: Umum & Branding -->
      <div v-if="activeTab === 'umum'" class="d-flex flex-column gap-3">
        <!-- Umum -->
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-settings me-2"></i>Umum</h3>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <div>
              <label class="form-label">Nama Aplikasi <span class="text-danger">*</span></label>
              <input v-model="form.app_name" class="form-control" :class="{ 'is-invalid': formErrors.app_name }" placeholder="CBT Patra" />
              <div v-if="formErrors.app_name" class="invalid-feedback">{{ formErrors.app_name }}</div>
            </div>
            <div>
              <label class="form-label">Metode Login</label>
              <select v-model="form.login_method" class="form-select">
                <option value="normal">Normal (Database)</option>
                <option value="redis">Redis (Cache Warm-up)</option>
              </select>
            </div>
            <div class="d-flex align-items-center justify-content-between">
              <div>
                <div class="fw-medium">WebSocket Real-time</div>
                <div class="text-muted small">Aktifkan monitoring real-time via WebSocket</div>
              </div>
              <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox"
                  :checked="form.websocket_enabled === '1'"
                  @change="form.websocket_enabled = ($event.target as HTMLInputElement).checked ? '1' : '0'" />
              </div>
            </div>
          </div>
        </div>

        <!-- Branding & Tampilan -->
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-palette me-2"></i>Branding & Tampilan</h3>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <!-- School Name -->
            <div>
              <label class="form-label">Nama Sekolah / Institusi</label>
              <input v-model="form.school_name" class="form-control" placeholder="SMA Negeri 1 Kota" />
              <div class="form-hint">Tampil di kartu peserta dan laporan</div>
            </div>

            <!-- Logo Upload -->
            <div>
              <label class="form-label">Logo Aplikasi</label>
              <div class="d-flex align-items-center gap-3">
                <div v-if="form.app_logo" class="avatar avatar-lg" style="border: 1px solid #e2e8f0;">
                  <img :src="form.app_logo" alt="Logo" />
                </div>
                <div v-else class="avatar avatar-lg bg-primary-lt">
                  <i class="ti ti-photo"></i>
                </div>
                <div class="flex-fill">
                  <input type="file" class="form-control form-control-sm" accept=".png,.jpg,.jpeg,.svg,.webp"
                    @change="handleBrandingUpload($event, 'app_logo')" :disabled="uploadingBranding" />
                  <div class="form-hint">PNG, JPG, SVG atau WebP. Max 2MB. Rekomendasi: 200x200px</div>
                </div>
              </div>
            </div>

            <!-- Favicon Upload -->
            <div>
              <label class="form-label">Favicon</label>
              <div class="d-flex align-items-center gap-3">
                <div v-if="form.app_favicon" class="avatar avatar-sm" style="border: 1px solid #e2e8f0;">
                  <img :src="form.app_favicon" alt="Favicon" />
                </div>
                <div v-else class="avatar avatar-sm bg-secondary-lt">
                  <i class="ti ti-world"></i>
                </div>
                <div class="flex-fill">
                  <input type="file" class="form-control form-control-sm" accept=".png,.ico,.svg"
                    @change="handleBrandingUpload($event, 'app_favicon')" :disabled="uploadingBranding" />
                  <div class="form-hint">ICO, PNG atau SVG. Max 2MB. Rekomendasi: 32x32px</div>
                </div>
              </div>
            </div>

            <!-- Footer Text -->
            <div>
              <label class="form-label">Teks Footer</label>
              <input v-model="form.app_footer_text" class="form-control" placeholder="Otomatis dari nama aplikasi jika kosong" />
            </div>

            <!-- Login Subtitle -->
            <div>
              <label class="form-label">Subtitle Halaman Login</label>
              <input v-model="form.login_subtitle" class="form-control" placeholder="Masuk ke akun Anda untuk melanjutkan" />
            </div>

            <!-- Primary Color -->
            <div>
              <label class="form-label">Warna Utama (Primary Color)</label>
              <div class="d-flex align-items-center gap-2">
                <input type="color" v-model="form.app_primary_color" class="form-control form-control-color" style="width:48px;height:36px" />
                <span class="font-monospace small text-muted">{{ form.app_primary_color || 'Default' }}</span>
                <button v-if="form.app_primary_color" class="btn btn-ghost-secondary btn-sm" @click="form.app_primary_color = ''">
                  <i class="ti ti-x"></i> Reset
                </button>
              </div>
              <div class="form-hint">Warna utama aplikasi (header, tombol, link). Kosongkan untuk default</div>
            </div>

            <!-- Login Background Image -->
            <div>
              <label class="form-label">Gambar Background Login</label>
              <div class="d-flex align-items-center gap-3">
                <div v-if="form.login_bg_image" class="rounded" style="width:64px;height:40px;overflow:hidden;border:1px solid #e2e8f0;">
                  <img :src="form.login_bg_image" alt="" style="width:100%;height:100%;object-fit:cover" />
                </div>
                <div class="flex-fill">
                  <input type="file" class="form-control form-control-sm" accept=".png,.jpg,.jpeg,.webp"
                    @change="handleBrandingUpload($event, 'login_bg_image')" :disabled="uploadingBranding" />
                  <div class="form-hint">JPG, PNG atau WebP. Max 2MB</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: IP Whitelist -->
      <div v-if="activeTab === 'ip-whitelist'">
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-shield-lock me-2"></i>IP Whitelist</h3>
            <div class="card-options">
              <BaseButton variant="primary" size="sm" :loading="savingWhitelist" @click="handleSaveWhitelist">
                <i class="ti ti-device-floppy me-1"></i>Simpan
              </BaseButton>
            </div>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <div class="alert alert-info mb-0">
              <div class="d-flex align-items-start gap-2">
                <i class="ti ti-info-circle fs-3 mt-1"></i>
                <div>
                  <div class="fw-bold">Tentang IP Whitelist</div>
                  <div class="small">
                    Jika diaktifkan, hanya perangkat dari IP yang terdaftar yang dapat mengakses ujian.
                    Admin selalu diizinkan mengakses tanpa batasan IP.
                  </div>
                </div>
              </div>
            </div>

            <div class="d-flex align-items-center justify-content-between">
              <div>
                <div class="fw-medium">Aktifkan IP Whitelist</div>
                <div class="text-muted small">Blokir akses ujian dari IP yang tidak terdaftar</div>
              </div>
              <div class="form-check form-switch mb-0">
                <input class="form-check-input" type="checkbox" v-model="ipWhitelistEnabled" />
              </div>
            </div>

            <div v-if="ipWhitelistEnabled">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <label class="form-label mb-0">Daftar IP / CIDR yang Diizinkan</label>
                <div class="d-flex gap-1">
                  <button class="btn btn-sm btn-ghost-secondary" :disabled="loadingMyIP" @click="detectMyIP">
                    <span v-if="loadingMyIP" class="spinner-border spinner-border-sm me-1"></span>
                    <i v-else class="ti ti-world-search me-1"></i>Deteksi IP Saya
                  </button>
                  <button v-if="myIP" class="btn btn-sm btn-ghost-primary" @click="addMyIP">
                    <i class="ti ti-plus me-1"></i>Tambah {{ myIP }}
                  </button>
                </div>
              </div>
              <textarea
                v-model="ipWhitelistIPs"
                class="form-control font-monospace"
                rows="8"
                placeholder="Masukkan satu IP/CIDR per baris, contoh:
192.168.1.0/24
10.0.0.1
172.16.0.0/16
203.0.113.50"
              ></textarea>
              <div class="form-hint mt-1">
                Satu IP/CIDR per baris. Mendukung format: <code>192.168.1.1</code> (IP tunggal) atau <code>192.168.1.0/24</code> (CIDR range).
              </div>

              <div class="alert alert-warning mt-3 mb-0">
                <div class="d-flex align-items-start gap-2">
                  <i class="ti ti-alert-triangle fs-3 mt-1"></i>
                  <div>
                    <div class="fw-bold">Perhatian</div>
                    <ul class="small mb-0 ps-3">
                      <li>Pastikan IP sekolah/lab komputer sudah terdaftar sebelum mengaktifkan</li>
                      <li>Admin selalu bisa mengakses tanpa batasan IP</li>
                      <li>Perubahan berlaku dalam 2 menit (cache TTL)</li>
                      <li>IP whitelist hanya mempengaruhi akses ujian peserta</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: Konfigurasi AI -->
      <div v-if="activeTab === 'ai'">
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-robot me-2"></i>Konfigurasi AI</h3>
            <div class="card-options">
              <button class="btn btn-sm btn-ghost-secondary" :disabled="testingAI" @click="handleTestAI">
                <span v-if="testingAI" class="spinner-border spinner-border-sm me-1"></span>
                <i v-else class="ti ti-flask me-1"></i>Test Koneksi AI
              </button>
            </div>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <div>
              <label class="form-label">AI API URL</label>
              <input v-model="form.ai_api_url" class="form-control" placeholder="https://api.openai.com/v1" />
            </div>
            <div>
              <label class="form-label">AI API Key</label>
              <input v-model="form.ai_api_key" class="form-control" type="password" placeholder="sk-..." />
            </div>
            <div>
              <label class="form-label">Header Autentikasi</label>
              <input v-model="form.ai_api_header" class="form-control" placeholder="Authorization" />
            </div>
            <div>
              <label class="form-label">Parameter Model (JSON)</label>
              <textarea v-model="form.ai_model_params" class="form-control" placeholder='{"model":"gpt-4","max_tokens":500}' rows="3"></textarea>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: Backup & Restore -->
      <div v-if="activeTab === 'backup'">
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-database me-2"></i>Backup & Restore Lokal</h3>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <p class="text-muted small mb-0">Ekspor pengaturan aplikasi ke file JSON, atau pulihkan dari file backup sebelumnya.</p>
            <div>
              <BaseButton variant="ghost" :loading="backingUp" @click="handleBackup">
                <i class="ti ti-database-export me-1"></i>Buat Backup Lokal
              </BaseButton>
            </div>
            <div>
              <label class="form-label">Restore dari File</label>
              <input ref="restoreFileInput" type="file" accept=".json" class="form-control"
                @change="handleRestoreFile" :disabled="restoring" />
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: Manajemen Database -->
      <div v-if="activeTab === 'database'">
        <div class="card cursor-pointer" @click="router.push({ name: 'DatabaseManagement' })" style="cursor: pointer;">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-database-cog me-2"></i>Manajemen Database</h3>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <p class="text-muted small mb-0">Export, import, dan kelola backup database sistem PostgreSQL.</p>
            <div class="d-flex align-items-center gap-3 text-muted">
              <div class="d-flex align-items-center gap-1">
                <i class="ti ti-database-export"></i>
                <span class="small">Export</span>
              </div>
              <div class="d-flex align-items-center gap-1">
                <i class="ti ti-database-import"></i>
                <span class="small">Import</span>
              </div>
              <div class="d-flex align-items-center gap-1">
                <i class="ti ti-history"></i>
                <span class="small">Riwayat</span>
              </div>
            </div>
            <div>
              <BaseButton variant="ghost" @click.stop="router.push({ name: 'DatabaseManagement' })">
                <i class="ti ti-arrow-right me-1"></i>Kelola Database
              </BaseButton>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: MinIO Cloud -->
      <div v-if="activeTab === 'minio'">
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-cloud me-2"></i>Backup MinIO (Object Storage)</h3>
            <div class="card-options">
              <button class="btn btn-sm btn-ghost-secondary" :disabled="testingMinio" @click="handleTestMinio">
                <span v-if="testingMinio" class="spinner-border spinner-border-sm me-1"></span>
                <i v-else class="ti ti-plug me-1"></i>Test Koneksi
              </button>
            </div>
          </div>
          <div class="card-body">
            <div class="row g-3">
              <!-- MinIO Config Form -->
              <div class="col-md-6">
                <div class="d-flex flex-column gap-3">
                  <div>
                    <label class="form-label">Endpoint MinIO</label>
                    <input v-model="minioForm.minio_endpoint" class="form-control"
                      placeholder="minio.example.com:9000 atau s3.amazonaws.com" />
                    <div class="form-hint">Tanpa protokol (http/https). Contoh: <code>minio.sekolah.sch.id:9000</code></div>
                  </div>
                  <div>
                    <label class="form-label">Nama Bucket</label>
                    <input v-model="minioForm.minio_bucket" class="form-control" placeholder="cbt-patra-backups" />
                  </div>
                  <div>
                    <label class="form-label">Access Key</label>
                    <input v-model="minioForm.minio_access_key" class="form-control" placeholder="minioadmin" />
                  </div>
                  <div>
                    <label class="form-label">Secret Key</label>
                    <input v-model="minioForm.minio_secret_key" class="form-control" type="password" placeholder="••••••••" />
                  </div>
                  <div class="d-flex align-items-center justify-content-between">
                    <div>
                      <div class="fw-medium">Gunakan SSL (HTTPS)</div>
                      <div class="text-muted small">Aktifkan jika endpoint menggunakan HTTPS</div>
                    </div>
                    <div class="form-check form-switch mb-0">
                      <input class="form-check-input" type="checkbox"
                        :checked="minioForm.minio_use_ssl === '1'"
                        @change="minioForm.minio_use_ssl = ($event.target as HTMLInputElement).checked ? '1' : '0'" />
                    </div>
                  </div>
                  <div>
                    <BaseButton variant="primary" :loading="savingMinio" @click="handleSaveMinio">
                      <i class="ti ti-device-floppy me-1"></i>Simpan Konfigurasi MinIO
                    </BaseButton>
                  </div>
                </div>
              </div>

              <!-- MinIO Actions -->
              <div class="col-md-6">
                <div class="d-flex flex-column gap-3">
                  <div class="card bg-blue-lt">
                    <div class="card-body">
                      <h4 class="card-title mb-2"><i class="ti ti-cloud-upload me-1"></i>Upload Backup ke MinIO</h4>
                      <p class="text-muted small mb-3">Buat file backup pengaturan dan upload langsung ke bucket MinIO.</p>
                      <BaseButton variant="primary" :loading="backingUpMinio" @click="handleBackupToMinio">
                        <i class="ti ti-cloud-upload me-1"></i>Backup ke MinIO
                      </BaseButton>
                    </div>
                  </div>

                  <div class="card bg-green-lt">
                    <div class="card-body">
                      <h4 class="card-title mb-2"><i class="ti ti-cloud-download me-1"></i>Restore dari MinIO</h4>
                      <p class="text-muted small mb-2">Ambil file backup dari MinIO dan pulihkan pengaturan aplikasi.</p>
                      <button class="btn btn-ghost-secondary mb-3" :disabled="loadingMinioList" @click="handleListMinioBackups">
                        <span v-if="loadingMinioList" class="spinner-border spinner-border-sm me-1"></span>
                        <i v-else class="ti ti-list me-1"></i>
                        Tampilkan Daftar Backup
                      </button>

                      <div v-if="showMinioBackupList">
                        <div v-if="minioBackups.length === 0" class="text-muted small">
                          Tidak ada file backup di MinIO.
                        </div>
                        <div v-else>
                          <label class="form-label">Pilih File Backup</label>
                          <select v-model="selectedMinioFile" class="form-select mb-2">
                            <option value="">-- Pilih backup --</option>
                            <option v-for="f in minioBackups" :key="f.key" :value="f.key">
                              {{ f.key }} ({{ formatBytes(f.size) }}) — {{ formatDate(f.last_modified) }}
                            </option>
                          </select>
                          <button
                            class="btn btn-success"
                            :disabled="restoringMinio || !selectedMinioFile"
                            @click="handleRestoreFromMinio"
                          >
                            <span v-if="restoringMinio" class="spinner-border spinner-border-sm me-1"></span>
                            <i v-else class="ti ti-restore me-1"></i>
                            Restore dari MinIO
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: PWA -->
      <div v-if="activeTab === 'pwa'">
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-device-mobile me-2"></i>Pengaturan PWA (Progressive Web App)</h3>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <p class="text-muted small mb-0">Atur tampilan saat aplikasi diinstal ke perangkat pengguna (Mobile/Desktop).</p>
            <div>
              <label class="form-label">Nama Aplikasi (PWA)</label>
              <input v-model="form.pwa_app_name" class="form-control" placeholder="CBT Patra App" />
            </div>
            <div>
              <label class="form-label">Nama Pendek (Short Name)</label>
              <input v-model="form.pwa_short_name" class="form-control" placeholder="Patra" />
            </div>
            <div class="row">
              <div class="col-6">
                <label class="form-label">Theme Color</label>
                <div class="d-flex align-items-center gap-2">
                  <input type="color" v-model="form.pwa_theme_color" class="form-control form-control-color" style="width:48px;height:36px" />
                  <span class="font-monospace small text-muted">{{ form.pwa_theme_color || '#ffffff' }}</span>
                </div>
              </div>
              <div class="col-6">
                <label class="form-label">Background Color</label>
                <div class="d-flex align-items-center gap-2">
                  <input type="color" v-model="form.pwa_background_color" class="form-control form-control-color" style="width:48px;height:36px" />
                  <span class="font-monospace small text-muted">{{ form.pwa_background_color || '#ffffff' }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab: Panic Mode -->
      <div v-if="activeTab === 'panic'">
        <div class="card" :class="panicModeActive ? 'border-danger' : ''">
          <div class="card-header" :class="panicModeActive ? 'bg-danger-subtle' : ''">
            <h3 class="card-title">
              <i class="ti ti-alert-triangle me-2" :class="panicModeActive ? 'text-danger' : ''"></i>
              Panic Mode
            </h3>
            <div class="card-options">
              <span v-if="panicModeActive" class="badge bg-danger-lt fw-semibold">AKTIF</span>
              <span v-else class="badge bg-success-lt fw-semibold">NON-AKTIF</span>
            </div>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <div v-if="panicModeActive" class="alert alert-danger mb-0">
              <div class="d-flex align-items-center gap-2">
                <i class="ti ti-shield-x fs-3 text-danger"></i>
                <div>
                  <div class="fw-bold">Panic Mode sedang AKTIF</div>
                  <div class="small">Semua akses peserta dikunci. Ujian tidak dapat dimulai atau dilanjutkan.</div>
                </div>
              </div>
            </div>
            <div v-else class="alert alert-secondary mb-0">
              <div class="d-flex align-items-center gap-2">
                <i class="ti ti-shield-check fs-3 text-muted"></i>
                <div class="small">Panic Mode tidak aktif. Aktifkan untuk memblokir seluruh akses peserta secara darurat.</div>
              </div>
            </div>

            <div class="d-flex gap-2">
              <BaseButton
                v-if="!panicModeActive"
                variant="danger"
                :loading="panicModeLoading"
                class="flex-fill"
                @click="askTogglePanicMode"
              >
                <i class="ti ti-lock me-1"></i>Aktifkan Panic Mode
              </BaseButton>
              <button
                v-else
                class="btn btn-success flex-fill"
                :disabled="panicModeLoading"
                @click="askTogglePanicMode"
              >
                <span v-if="panicModeLoading" class="spinner-border spinner-border-sm me-1"></span>
                <i v-else class="ti ti-lock-open me-1"></i>
                Nonaktifkan Panic Mode
              </button>
            </div>
            <p class="text-muted small mb-0">
              <i class="ti ti-info-circle me-1"></i>
              Perubahan berlaku instan tanpa perlu menyimpan pengaturan.
            </p>
          </div>
        </div>
      </div>

      <!-- Tab: Cache & Performa -->
      <div v-if="activeTab === 'cache'">
        <div class="card">
          <div class="card-header">
            <h3 class="card-title"><i class="ti ti-bolt me-2"></i>Cache & Performa</h3>
          </div>
          <div class="card-body d-flex flex-column gap-3">
            <div class="d-flex align-items-center justify-content-between">
              <div>
                <div class="fw-medium">Bersihkan Cache Sistem</div>
                <div class="text-muted small">Bersihkan memori Redis atau Cache sistem. Peserta mungkin akan logout setelah cache dibersihkan.</div>
              </div>
              <div>
                <BaseButton variant="danger" :loading="clearingCache" @click="askClearCache">
                  <i class="ti ti-trash me-1"></i>Bersihkan Cache
                </BaseButton>
              </div>
            </div>
            <hr class="my-0">
            <div class="alert alert-warning mb-0">
              <div class="d-flex align-items-center gap-2">
                <i class="ti ti-alert-circle fs-3"></i>
                <div class="small">
                  Membersihkan cache akan menghapus seluruh data sesi yang tersimpan di Redis. Peserta yang sedang mengerjakan ujian akan diminta untuk login ulang.
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>

  <BaseConfirmModal
    v-if="showPanicConfirm"
    :title="panicModeActive ? 'Nonaktifkan Panic Mode' : 'Aktifkan Panic Mode'"
    :message="panicModeActive ? 'Nonaktifkan Panic Mode? Peserta akan bisa melanjutkan ujian.' : 'Aktifkan Panic Mode? Semua akses peserta akan dikunci secara instan.'"
    :confirm-label="panicModeActive ? 'Ya, Nonaktifkan' : 'Ya, Aktifkan'"
    :confirm-variant="panicModeActive ? 'primary' : 'danger'"
    @confirm="handleTogglePanicMode"
    @close="showPanicConfirm = false"
  />

  <BaseConfirmModal
    v-if="showClearCacheConfirm"
    title="Bersihkan Cache"
    message="Bersihkan seluruh cache sistem? Peserta mungkin akan logout."
    confirm-label="Ya, Bersihkan"
    confirm-variant="danger"
    @confirm="handleClearCache"
    @close="showClearCacheConfirm = false"
  />
</template>

<style scoped>
.settings-mobile-nav {
  display: flex;
  overflow-x: auto;
  gap: 0.25rem;
  scrollbar-width: none;
  -ms-overflow-style: none;
  -webkit-overflow-scrolling: touch;
}

.settings-mobile-nav::-webkit-scrollbar {
  display: none;
}

.settings-mobile-nav-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.75rem;
  border-radius: 6px;
  white-space: nowrap;
  font-size: 0.8125rem;
  color: var(--tblr-body-color);
  text-decoration: none;
  background: transparent;
  transition: background 0.15s, color 0.15s;
  border: 1px solid transparent;
}

.settings-mobile-nav-item:hover {
  background: var(--tblr-bg-surface-secondary);
  color: var(--tblr-body-color);
  text-decoration: none;
}

.settings-mobile-nav-item.active {
  background: var(--tblr-primary);
  color: #fff;
  border-color: var(--tblr-primary);
}

.settings-mobile-nav-item.active .ti {
  color: #fff !important;
}
</style>
