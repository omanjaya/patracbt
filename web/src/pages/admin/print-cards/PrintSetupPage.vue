<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import client from '../../../api/client'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const router = useRouter()

const rombels = ref<Rombel[]>([])
const loadingRombels = ref(true)
const loadingSettings = ref(true)
const showModal = ref(false)

const form = ref({
  rombel_id: '',
  sign_title: 'Kepala Sekolah',
  sign_name: '',
  sign_nip: '',
})

const STORAGE_KEY = 'print_card_config'

function loadFromStorage() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      const config = JSON.parse(stored)
      if (config.title) form.value.sign_title = config.title
      if (config.name) form.value.sign_name = config.name
      if (config.nip) form.value.sign_nip = config.nip
    }
  } catch {
    // ignore
  }
}

function saveToStorage() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({
    title: form.value.sign_title,
    name: form.value.sign_name,
    nip: form.value.sign_nip,
  }))
}

async function loadCardSettings() {
  loadingSettings.value = true
  try {
    const res = await client.get('/admin/cards/settings')
    const data = res.data.data
    // Pre-fill from server settings if local storage is empty
    const stored = localStorage.getItem(STORAGE_KEY)
    if (!stored) {
      if (data?.headmaster_name) form.value.sign_name = data.headmaster_name
      if (data?.headmaster_nip) form.value.sign_nip = data.headmaster_nip
    }
  } catch {
    // Non-critical, use localStorage fallback
  } finally {
    loadingSettings.value = false
  }
}

function openModal() {
  loadFromStorage()
  showModal.value = true
}

function handleSubmit() {
  saveToStorage()
  // Navigate to print cards page with query params
  router.push({
    path: '/admin/print-cards',
    query: {
      rombel_id: form.value.rombel_id || undefined,
      sign_title: form.value.sign_title,
      sign_name: form.value.sign_name,
      sign_nip: form.value.sign_nip || undefined,
    },
  })
}

onMounted(async () => {
  loadFromStorage()

  // Load rombels and card settings in parallel
  loadingRombels.value = true
  const [rombelRes] = await Promise.all([
    rombelApi.list({ per_page: 200 }).finally(() => { loadingRombels.value = false }),
    loadCardSettings(),
  ])
  rombels.value = rombelRes.data.data ?? []
})
</script>

<template>
  <BasePageHeader
    title="Konfigurasi Cetak Kartu"
    subtitle="Atur data penanda tangan sebelum mencetak kartu peserta ujian"
    :breadcrumbs="[{ label: 'Kartu Peserta', to: '/admin/print-cards' }, { label: 'Setup Cetak' }]"
  />

  <div class="row row-cards justify-content-center">
    <div class="col-md-6 col-lg-5">
      <div class="card">
        <div class="card-status-top bg-primary"></div>
        <div class="card-body text-center py-5">
          <div class="mb-4">
            <span class="avatar avatar-xl bg-primary-lt rounded-circle">
              <i class="ti ti-printer" style="font-size: 2.2rem;"></i>
            </span>
          </div>
          <h3 class="mb-2">Siap Mencetak Kartu Peserta?</h3>
          <p class="text-muted mb-4">
            Silakan atur data penanda tangan (Kepala Sekolah / Panitia) sebelum mencetak.
            Data ini akan disimpan di browser Anda untuk penggunaan berikutnya.
          </p>
          <button type="button" class="btn btn-primary w-100" @click="openModal">
            <i class="ti ti-settings me-2"></i>Atur Data &amp; Cetak
          </button>
        </div>
      </div>
    </div>

    <!-- Info card -->
    <div class="col-md-6 col-lg-5">
      <div class="card h-100">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-info-circle me-2"></i>Panduan
          </h3>
        </div>
        <div class="card-body d-flex flex-column gap-3">
          <div class="d-flex gap-3">
            <span class="avatar avatar-sm bg-blue-lt text-blue flex-shrink-0">
              <i class="ti ti-filter"></i>
            </span>
            <div>
              <div class="fw-semibold small">Filter Rombel</div>
              <div class="text-muted small">Pilih rombel tertentu atau cetak semua peserta sekaligus.</div>
            </div>
          </div>
          <div class="d-flex gap-3">
            <span class="avatar avatar-sm bg-green-lt text-green flex-shrink-0">
              <i class="ti ti-writing"></i>
            </span>
            <div>
              <div class="fw-semibold small">Tanda Tangan</div>
              <div class="text-muted small">Data jabatan dan nama pejabat penanda tangan akan dicetak di kartu.</div>
            </div>
          </div>
          <div class="d-flex gap-3">
            <span class="avatar avatar-sm bg-orange-lt text-orange flex-shrink-0">
              <i class="ti ti-device-floppy"></i>
            </span>
            <div>
              <div class="fw-semibold small">Data Tersimpan Otomatis</div>
              <div class="text-muted small">Konfigurasi tersimpan di browser, tidak perlu isi ulang setiap kali cetak.</div>
            </div>
          </div>
          <div class="d-flex gap-3">
            <span class="avatar avatar-sm bg-purple-lt text-purple flex-shrink-0">
              <i class="ti ti-id-badge"></i>
            </span>
            <div>
              <div class="fw-semibold small">Format Kartu</div>
              <div class="text-muted small">Kartu memuat nama, username, NIS/NIP, dan kelas peserta.</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Modal Konfigurasi -->
  <div v-if="showModal" class="modal modal-blur show d-block" tabindex="-1" role="dialog">
    <div class="modal-dialog modal-dialog-centered" role="document">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">
            <i class="ti ti-settings me-2"></i>Pengaturan Cetak
          </h5>
          <button type="button" class="btn-close" @click="showModal = false"></button>
        </div>
        <div class="modal-body">

          <!-- Filter Rombel -->
          <div class="mb-3">
            <label class="form-label">
              <i class="ti ti-users-group me-1"></i>Filter Rombel <span class="text-muted">(Opsional)</span>
            </label>
            <select v-model="form.rombel_id" class="form-select">
              <option value="">Semua Rombel</option>
              <option
                v-for="rombel in rombels"
                :key="rombel.id"
                :value="String(rombel.id)"
              >
                {{ rombel.name }}
                <template v-if="rombel.grade_level"> — {{ rombel.grade_level }}</template>
              </option>
            </select>
            <div v-if="loadingRombels" class="form-hint mt-1 text-muted small">
              <span class="spinner-border spinner-border-sm me-1"></span>Memuat daftar rombel...
            </div>
          </div>

          <div class="hr-text">
            <i class="ti ti-writing me-1"></i>Tanda Tangan
          </div>

          <!-- Jabatan / Judul -->
          <div class="mb-3">
            <label class="form-label required">
              <i class="ti ti-briefcase me-1"></i>Jabatan / Judul
            </label>
            <input
              v-model="form.sign_title"
              type="text"
              class="form-control"
              placeholder="Contoh: Kepala Sekolah"
              required
            />
          </div>

          <!-- Nama Pejabat -->
          <div class="mb-3">
            <label class="form-label required">
              <i class="ti ti-user me-1"></i>Nama Pejabat
            </label>
            <input
              v-model="form.sign_name"
              type="text"
              class="form-control"
              placeholder="Nama lengkap beserta gelar"
              required
            />
          </div>

          <!-- NIP / NIY -->
          <div class="mb-3">
            <label class="form-label">
              <i class="ti ti-hash me-1"></i>NIP / NIY <span class="text-muted">(Opsional)</span>
            </label>
            <input
              v-model="form.sign_nip"
              type="text"
              class="form-control"
              placeholder="Nomor Induk Pegawai"
            />
          </div>

        </div>
        <div class="modal-footer">
          <button type="button" class="btn me-auto" @click="showModal = false">
            <i class="ti ti-x me-1"></i>Batal
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="!form.sign_title.trim() || !form.sign_name.trim()"
            @click="handleSubmit"
          >
            <i class="ti ti-printer me-1"></i>Print Preview
          </button>
        </div>
      </div>
    </div>
  </div>
  <div v-if="showModal" class="modal-backdrop fade show" @click="showModal = false"></div>
</template>
