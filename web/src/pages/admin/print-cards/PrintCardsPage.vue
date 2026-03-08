<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import client from '../../../api/client'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

interface CardStudent {
  name: string
  username: string
  password_plain: string
  nis: string | null
  rombel_name: string
  room_name: string
  avatar_url: string
  qr_base64?: string
}

interface CardSettings {
  app_name: string
  school_name: string
  logo_url: string
  header_color: string
  headmaster_name: string
  headmaster_nip: string
}

const route = useRoute()
const router = useRouter()

const students = ref<CardStudent[]>([])
const settings = ref<CardSettings | null>(null)
const loading = ref(true)
const error = ref('')

// Query params from PrintSetupPage
const signTitle = computed(() => (route.query.sign_title as string) || 'Kepala Sekolah')
const signName = computed(() => (route.query.sign_name as string) || '')
const signNip = computed(() => (route.query.sign_nip as string) || '')
const rombelId = computed(() => (route.query.rombel_id as string) || '')

// Chunk students into pages of 8
const pages = computed(() => {
  const result: CardStudent[][] = []
  for (let i = 0; i < students.value.length; i += 8) {
    result.push(students.value.slice(i, i + 8))
  }
  return result
})

const totalPages = computed(() => pages.value.length)

async function fetchCards() {
  loading.value = true
  error.value = ''
  try {
    const params: Record<string, string> = {}
    if (rombelId.value) params.rombel_id = rombelId.value

    const res = await client.get('/admin/cards/with-qr', { params })
    students.value = res.data.data?.students ?? []
    settings.value = res.data.data?.settings ?? null
  } catch (e: any) {
    error.value = e?.response?.data?.message ?? 'Gagal memuat data kartu peserta'
  } finally {
    loading.value = false
  }
}

function handlePrint() {
  const iframe = document.getElementById('print-frame') as HTMLIFrameElement
  if (iframe?.contentWindow) {
    iframe.contentWindow.focus()
    iframe.contentWindow.print()
  }
}

function goBack() {
  router.push('/admin/print-cards/setup')
}

// Shorten long names (keep first 4 words, abbreviate the rest)
function shortenName(name: string, limit = 25): string {
  if (name.length <= limit) return name
  const words = name.split(' ')
  if (words.length <= 4) return name
  const kept = words.slice(0, 4)
  const initials = words.slice(4).map(w => w.charAt(0) + '.').join('')
  return kept.join(' ') + ' ' + initials
}

// Build iframe HTML for print preview
const iframeHtml = computed(() => {
  if (!settings.value || !students.value.length) return ''

  const s = settings.value
  const headerColor = s.header_color || '#2c3e50'
  const schoolName = s.school_name || s.app_name || 'CBT PATRA'
  const logoUrl = s.logo_url || ''

  const styles = `
    <style>
      @page { size: A4; margin: 0; }
      * { box-sizing: border-box; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
      body {
        margin: 0; padding: 0;
        background-color: #525659;
        font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        display: flex; flex-direction: column; align-items: center;
        padding: 20px 0; gap: 30px;
      }
      .page-sheet {
        background: white; width: 210mm; height: 297mm;
        padding: 10mm;
        box-shadow: 0 4px 10px rgba(0,0,0,0.5);
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        grid-template-rows: repeat(4, 1fr);
        gap: 10mm;
        page-break-after: always; break-after: page;
      }
      .page-sheet:last-child { page-break-after: avoid; break-after: avoid; }

      .id-card {
        background: #fff; border: 1px solid #999;
        width: 100%; height: 60mm;
        position: relative; overflow: hidden;
        border-radius: 6px; color: #000;
      }
      .card-header {
        background-color: ${headerColor};
        color: white; padding: 24px 8px;
        display: flex; align-items: center;
        height: 32px;
        border-bottom: 2px solid #f1c40f;
      }
      .logo { width: 24px; height: 24px; object-fit: contain; margin-right: 8px; background: #fff; border-radius: 50%; padding: 1px; }
      .school-info h2 { margin: 0; padding: 0; font-size: 8pt; font-weight: 800; text-transform: uppercase; line-height: 1; }
      .school-info p { margin: 2px 0 0 0; padding: 0; font-size: 5pt; opacity: 0.9; line-height: 1; }

      .card-body { display: flex; padding: 6px; gap: 8px; }
      .left-col { width: 22mm; text-align: center; display: flex; flex-direction: column; align-items: center; }
      .photo-box {
        width: 20mm; height: 26mm; border-radius: 8px;
        border: 1px solid #ccc; background: #eee; margin-bottom: 2px;
      }
      .photo-box img { width: 100%; height: 100%; object-fit: cover; border-radius: 8px; display: block; }
      .qr-box { width: 14mm; height: 14mm; background: #fff; margin-top: 2px; }
      .qr-box svg, .qr-box img { width: 100%; height: 100%; display: block; }

      .right-col { flex: 1; font-size: 8pt; }
      .detail-row { display: flex; border-bottom: 1px dotted #ddd; margin-bottom: 2px; line-height: 1.3; }
      .label { width: 50px; color: #555; font-size: 7.5pt; }
      .sep { margin-right: 4px; }
      .value { flex: 1; font-weight: 700; color: #000; white-space: nowrap; }

      .login-box {
        margin-top: 4px; border: 1px dashed #444;
        background-color: #fffde7; padding: 2px 5px;
        border-radius: 3px; width: 90%;
      }
      .login-row { display: flex; font-family: 'Courier New', monospace; font-size: 8.5pt; font-weight: bold; margin: 0; }
      .login-lbl { width: 40px; color: #333; }

      .signature-area {
        position: absolute; bottom: 4px; right: 8px;
        text-align: center; font-size: 6.5pt; width: 35mm;
        z-index: 10; line-height: 1.1;
      }
      .sign-name { margin-top: 18px; font-weight: bold; text-decoration: underline; font-size: 7pt; }

      .watermark {
        position: absolute; top: 63%; left: 50%;
        transform: translate(-50%, -50%);
        width: 50%; opacity: 0.09; z-index: 1;
        pointer-events: none;
        display: flex; justify-content: center; align-items: center;
      }
      .watermark img { width: 100%; height: auto; }
      .watermark .wm-text { font-size: 30pt; font-weight: 900; color: #ccc; }

      @media print {
        body { background: none; display: block; padding: 0; margin: 0; }
        .page-sheet { margin: 0; box-shadow: none; border: none; width: 210mm; height: 297mm; page-break-after: always; }
      }
    </style>
  `

  let cardsHtml = ''
  for (const page of pages.value) {
    let pageHtml = '<div class="page-sheet">'
    for (const student of page) {
      const displayName = shortenName(student.name)
      const nameFontSize = displayName.length > 25 ? '7pt' : '8pt'

      pageHtml += `
        <div class="id-card">
          <div class="watermark">
            ${logoUrl ? `<img src="${logoUrl}" alt="Watermark">` : '<span class="wm-text">CBT</span>'}
          </div>
          <div class="card-header">
            ${logoUrl ? `<img src="${logoUrl}" class="logo">` : ''}
            <div class="school-info">
              <h2>${escapeHtml(schoolName)}</h2>
              <p>KARTU PESERTA UJIAN</p>
            </div>
          </div>
          <div class="card-body">
            <div class="left-col">
              <div class="photo-box">
                <img src="${student.avatar_url || ('https://ui-avatars.com/api/?name=' + encodeURIComponent(student.name) + '&background=ccc&color=fff&size=200')}" alt="Foto Peserta" style="width: 100%; height: 100%; object-fit: cover;">
              </div>
              ${student.qr_base64 ? `<div class="qr-box"><img src="${student.qr_base64}" alt="QR"></div>` : ''}
            </div>
            <div class="right-col">
              <div class="detail-row">
                <span class="label">Nama</span><span class="sep">:</span>
                <span class="value" style="font-size: ${nameFontSize};">${escapeHtml(displayName)}</span>
              </div>
              <div class="detail-row">
                <span class="label">NIS/NISN</span><span class="sep">:</span>
                <span class="value">${escapeHtml(student.nis || '-')}</span>
              </div>
              <div class="detail-row">
                <span class="label">Kelas</span><span class="sep">:</span>
                <span class="value">${escapeHtml(student.rombel_name || '-')}</span>
              </div>
              <div class="detail-row">
                <span class="label">Ruang</span><span class="sep">:</span>
                <span class="value">${escapeHtml(student.room_name || 'Lab 1')}</span>
              </div>
              <div class="login-box">
                <div class="login-row"><span class="login-lbl">User</span>: ${escapeHtml(student.username)}</div>
                <div class="login-row"><span class="login-lbl">Pass</span>: ${escapeHtml(student.password_plain || '***')}</div>
              </div>
            </div>
          </div>
          <div class="signature-area">
            <div style="margin-bottom: 2px;">${escapeHtml(signTitle.value)}</div>
            <div class="sign-name">${escapeHtml(signName.value)}</div>
            ${signNip.value && signNip.value !== '-' ? `<div>${escapeHtml(signNip.value)}</div>` : ''}
          </div>
        </div>
      `
    }
    pageHtml += '</div>'
    cardsHtml += pageHtml
  }

  return `<!DOCTYPE html><html><head><title>Preview Kartu Peserta</title>${styles}</head><body>${cardsHtml}</body></html>`
})

function escapeHtml(str: string): string {
  const div = document.createElement('div')
  div.textContent = str
  return div.innerHTML
}

// Inject HTML into iframe when data is ready
function injectIframe() {
  const iframe = document.getElementById('print-frame') as HTMLIFrameElement
  if (!iframe?.contentWindow) return
  const doc = iframe.contentWindow.document
  doc.open()
  doc.write(iframeHtml.value)
  doc.close()
}

onMounted(async () => {
  await fetchCards()
  // Wait for next tick to let computed update
  setTimeout(injectIframe, 100)
})
</script>

<template>
  <div class="d-print-none">
    <BasePageHeader
      title="Cetak Kartu Peserta"
      :subtitle="`${students.length} Siswa (${totalPages} Halaman A4)`"
      :breadcrumbs="[{ label: 'Kartu Peserta', to: '/admin/print-cards/setup' }, { label: 'Cetak' }]"
    >
      <template #actions>
        <button class="btn btn-outline-secondary" @click="goBack">
          <i class="ti ti-arrow-left me-1"></i>Kembali
        </button>
        <button
          class="btn btn-primary"
          :disabled="!students.length"
          @click="handlePrint"
        >
          <i class="ti ti-printer me-1"></i>Cetak Sekarang
        </button>
      </template>
    </BasePageHeader>

    <div v-if="loading" class="p-5 text-center text-muted">
      <span class="spinner-border spinner-border-sm me-2"></span>Memuat data kartu...
    </div>

    <div v-else-if="error" class="alert alert-danger">
      <i class="ti ti-alert-circle me-2"></i>{{ error }}
    </div>

    <div v-else-if="!students.length" class="card">
      <div class="card-body text-center py-5">
        <div class="mb-3">
          <span class="avatar avatar-xl bg-muted-lt rounded-circle">
            <i class="ti ti-users-minus" style="font-size: 2rem;"></i>
          </span>
        </div>
        <h3 class="text-muted">Belum ada peserta</h3>
        <p class="text-muted">Tidak ada data peserta yang ditemukan untuk rombel ini.</p>
        <button class="btn btn-primary" @click="goBack">
          <i class="ti ti-arrow-left me-1"></i>Kembali ke Setup
        </button>
      </div>
    </div>

    <template v-else>
      <div class="card">
        <div class="card-body p-0">
          <iframe
            id="print-frame"
            style="width: 100%; height: 80vh; border: none; display: block;"
          ></iframe>
        </div>
      </div>
    </template>
  </div>
</template>
