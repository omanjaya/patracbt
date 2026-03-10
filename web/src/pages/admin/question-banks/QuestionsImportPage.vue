<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { questionBankApi, type QuestionBank } from '../../../api/question_bank.api'
import { useToastStore } from '@/stores/toast.store'
import { getErrorMessage } from '@/utils/apiError'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const route = useRoute()
const router = useRouter()
const toastStore = useToastStore()
const bankId = Number(route.params.id)

const bank = ref<QuestionBank | null>(null)
const loadingBank = ref(true)
const content = ref('')
const processing = ref(false)
const copied = ref(false)
const downloadingTpl = ref(false)

// Real-time validation
const formErrors = reactive<Record<string, string>>({})

function validateImportField(field: string) {
  delete formErrors[field]
  if (field === 'content' && !content.value.trim()) {
    formErrors.content = 'Konten soal tidak boleh kosong'
  }
}

const TEMPLATE_TEXT = `--- KATEGORI 1: FORMAT STANDAR ---

Soal:1) Ibukota negara Indonesia saat ini adalah...
A. Bandung
B. Jakarta
C. Surabaya
D. Nusantara
Kunci:D

Soal:2) Manakah yang termasuk hewan mamalia?
A. Paus
B. Ayam
C. Kucing
D. Ikan
Kunci:A,C

Soal:3) [BENAR-SALAH] Matahari terbit dari arah barat.
A. Benar
B. Salah
Kunci:B

Soal:4) [MENJODOHKAN] Pasangkan negara berikut dengan ibukotanya.
A. Jepang = Tokyo
B. Inggris = London
C. Perancis = Paris

Soal:5) [ISIAN-SINGKAT] Siapakah Presiden pertama Indonesia?
Kunci:Soekarno, Ir. Soekarno, Bung Karno

Soal:6) [ESAI] Jelaskan dampak pemanasan global.
(Jawaban dinilai manual oleh guru)


--- KATEGORI 2: FORMAT LANJUTAN (CUSTOM BOBOT) ---

Soal:7) Soal pilihan ganda dengan bobot khusus?
Poin:2.5
A. [100%] Jawaban Tepat Sempurna
B. [50%] Jawaban Kurang Tepat
C. [0%] Jawaban Salah
D. [-25%] Jawaban Minus (Penalti)

Soal:8) [ISIAN-SINGKAT] Isian singkat dengan bobot khusus
Poin:2
Kunci:Jakarta=100%, DKI Jakarta=80%, Djakarta=50%

Soal:9) [MATRIX] Tentukan fakta atau mitos.
Poin:5
Kolom: Fakta, Mitos
Baris: Bumi berbentuk bulat = 1
Baris: Manusia hanya memakai 10% otak = 2
(Angka 1 = Kolom Pertama, Angka 2 = Kolom Kedua)

--- KATEGORI FORMAT WACANA (STIMULUS) ---

[WACANA]
Topik: Berita Pembangunan Transportasi
KOTA NUSANTARA — Pemerintah resmi meluncurkan layanan Kereta Cepat Nusantara Raya (KCNR) pada Senin (21/10). Proyek ini diklaim akan memangkas waktu tempuh antara Kota Nusantara–Balikpapan dari 3 jam menjadi hanya 45 menit.

Soal:1) Faktor yang menjadi tujuan pembangunan KCNR adalah…
A. Mengurangi emisi karbon
B. Mempercepat waktu perjalanan
C. Meningkatkan harga tiket transportasi udara
D. Menambah jalur ke Samarinda
Kunci: A, B, D

Soal:2) [ISIAN-SINGKAT] Berapa lama waktu tempuh Kota Nusantara–Balikpapan dengan KCNR?
Kunci: 45 menit, 45, ±45 menit`

async function fetchBank() {
  try {
    const res = await questionBankApi.getById(bankId)
    bank.value = res.data.data
  } finally {
    loadingBank.value = false
  }
}

import client from '../../../api/client'

async function downloadTemplate() {
  downloadingTpl.value = true
  try {
    const res = await client.get(`/question-banks/${bankId}/import/template`, { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([res.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', 'template-import-soal.csv')
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  } catch (e) {
    toastStore.error(getErrorMessage(e, 'Gagal mengunduh template'))
  } finally {
    downloadingTpl.value = false
  }
}

async function handleSubmit() {
  if (!content.value.trim()) {
    toastStore.error('Konten soal tidak boleh kosong. Silakan tempel soal terlebih dahulu.')
    return
  }
  processing.value = true
  try {
    await questionBankApi.importQuestions(bankId, { content: content.value })
    toastStore.success('Soal berhasil diimport')
    router.push(`/admin/question-banks/${bankId}`)
  } catch (e) {
    toastStore.error(getErrorMessage(e, 'Gagal mengimport soal'))
  } finally {
    processing.value = false
  }
}

function copyTemplate() {
  const clean = TEMPLATE_TEXT.replace(/--- KATEGORI.*---/g, '').trim()
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(clean).then(() => showCopied())
  } else {
    const el = document.createElement('textarea')
    el.value = clean
    el.style.position = 'fixed'
    el.style.left = '-9999px'
    document.body.appendChild(el)
    el.focus()
    el.select()
    document.execCommand('copy')
    document.body.removeChild(el)
    showCopied()
  }
}

function showCopied() {
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

/** Clean Word/MSO HTML from clipboard paste */
function cleanWordHTML(html: string): string {
  let s = html
  // Remove HTML comments
  s = s.replace(/<!--[\s\S]*?-->/g, '')
  // Remove <style> blocks
  s = s.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
  // Remove mso-* CSS properties
  s = s.replace(/\s*mso-[^;:"]+:[^;:"]+;?/gi, '')
  // Remove class="Mso*" attributes
  s = s.replace(/\s+class="Mso[^"]*"/gi, '')
  // Remove empty style attributes
  s = s.replace(/\s+style="\s*"/gi, '')
  // Remove empty spans and divs
  s = s.replace(/<span[^>]*>\s*<\/span>/gi, '')
  s = s.replace(/<div[^>]*>\s*<\/div>/gi, '')
  // Normalize <br> to newline
  s = s.replace(/<br\s*\/?>/gi, '\n')
  // Convert </p> to newline, remove <p>
  s = s.replace(/<p[^>]*>/gi, '')
  s = s.replace(/<\/p>/gi, '\n')
  // Remove remaining HTML tags
  s = s.replace(/<[^>]+>/g, '')
  // Decode HTML entities
  s = s.replace(/&nbsp;/g, ' ')
  s = s.replace(/&amp;/g, '&')
  s = s.replace(/&lt;/g, '<')
  s = s.replace(/&gt;/g, '>')
  s = s.replace(/&quot;/g, '"')
  // Collapse excessive whitespace on lines
  s = s.replace(/[ \t]+/g, ' ')
  // Collapse 3+ newlines to 2
  s = s.replace(/\n{3,}/g, '\n\n')
  return s.trim()
}

function onPaste(e: ClipboardEvent) {
  const html = e.clipboardData?.getData('text/html')
  if (html && html.includes('mso-') || html && html.includes('MsoNormal')) {
    e.preventDefault()
    const cleaned = cleanWordHTML(html)
    // Insert at cursor or replace selection
    const textarea = e.target as HTMLTextAreaElement
    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const before = content.value.substring(0, start)
    const after = content.value.substring(end)
    content.value = before + cleaned + after
  }
}

onMounted(() => {
  fetchBank()
  // Load AI-generated content from sessionStorage if available
  const aiContent = sessionStorage.getItem('ai_generated_content')
  if (aiContent) {
    content.value = aiContent
    sessionStorage.removeItem('ai_generated_content')
    sessionStorage.removeItem('ai_generated_source')
  }
})
</script>

<template>
  <BasePageHeader
    title="Import Soal (Copy-Paste)"
    :subtitle="bank ? bank.name : 'Memuat...'"
    :breadcrumbs="[{ label: 'Bank Soal', to: '/admin/question-banks' }, { label: 'Import Soal' }]"
  >
    <template #actions>
      <button class="btn btn-outline-secondary" @click="downloadTemplate" :disabled="downloadingTpl">
        <span v-if="downloadingTpl" class="spinner-border spinner-border-sm me-1"></span>
        <i v-else class="ti ti-download me-1"></i>Download Template CSV
      </button>
      <button class="btn btn-outline-primary" @click="router.push(`/admin/question-banks/${bankId}`)">
        <i class="ti ti-edit me-1"></i>Kembali ke Editor Soal
      </button>
    </template>
  </BasePageHeader>

  <div class="row g-4">

    <!-- Editor Area -->
    <div class="col-lg-7">
      <div class="card h-100">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-clipboard-text me-2"></i>Area Editor
          </h3>
        </div>
        <div class="card-body d-flex flex-column gap-3">

          <div>
            <label class="form-label">Bank Soal Tujuan</label>
            <div v-if="loadingBank" class="placeholder-glow">
              <span class="placeholder col-8 rounded"></span>
            </div>
            <input
              v-else
              type="text"
              class="form-control"
              :value="bank ? `${bank.name}${bank.subject ? ' (' + bank.subject.name + ')' : ''}` : ''"
              disabled
            />
          </div>

          <div class="flex-grow-1 d-flex flex-column">
            <label class="form-label required">Tempel (Paste) Soal di Sini</label>
            <textarea
              v-model="content"
              class="form-control flex-grow-1"
              :class="{ 'is-invalid': formErrors.content }"
              rows="18"
              placeholder="Tempel teks soal di sini. Gunakan format template yang tersedia di panel kanan. Mendukung paste dari Microsoft Word."
              @blur="validateImportField('content')"
              @input="formErrors.content = ''"
              @paste="onPaste"
            ></textarea>
            <div v-if="formErrors.content" class="invalid-feedback">{{ formErrors.content }}</div>
            <div class="form-hint mt-2">
              <i class="ti ti-info-circle me-1"></i>
              Tips: Anda bisa menyalin tabel, gambar, dan teks langsung dari Microsoft Word.
            </div>
          </div>
        </div>
        <div class="card-footer text-end">
          <button
            type="button"
            class="btn btn-primary"
            :disabled="processing || !content.trim()"
            @click="handleSubmit"
          >
            <span v-if="processing" class="spinner-border spinner-border-sm me-2" role="status"></span>
            <i v-else class="ti ti-upload me-1"></i>
            <span>{{ processing ? 'Memproses...' : 'Proses Import' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Panduan Format -->
    <div class="col-lg-5">
      <div class="card h-100">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-book me-2"></i>Panduan Format Soal
          </h3>
          <div class="card-actions">
            <button class="btn btn-sm btn-ghost-primary" @click="copyTemplate">
              <i :class="copied ? 'ti ti-check text-success' : 'ti ti-copy'"></i>
              <span class="ms-1">{{ copied ? 'Tersalin!' : 'Salin Template' }}</span>
            </button>
          </div>
        </div>
        <div class="card-body d-flex flex-column gap-3">

          <div class="alert alert-info mb-0">
            <div class="d-flex gap-2">
              <i class="ti ti-alert-circle flex-shrink-0 mt-1"></i>
              <ul class="mb-0 ps-0" style="list-style: none;">
                <li class="mb-1">Setiap soal <strong>wajib</strong> diawali kata <code>Soal:[angka]</code></li>
                <li class="mb-1">Sistem otomatis mendeteksi tipe soal (PG, Esai, dll).</li>
                <li>Anda dapat menyertakan gambar di dalam teks soal.</li>
              </ul>
            </div>
          </div>

          <div>
            <label class="form-label fw-semibold">Template Siap Pakai</label>
            <textarea
              class="form-control"
              rows="24"
              readonly
              :value="TEMPLATE_TEXT"
              style="font-family: monospace; font-size: 0.82rem;"
            ></textarea>
            <div class="form-hint mt-2 text-muted">
              <i class="ti ti-asterisk me-1"></i>
              Tip: Salin teks di atas, tempel ke editor, lalu ubah isinya sesuai kebutuhan soal Anda.
            </div>
          </div>

        </div>
      </div>
    </div>

  </div>
</template>
