<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { settingApi } from '../../../api/setting.api'
import { useToastStore } from '../../../stores/toast.store'
import BaseConfirmModal from '@/components/ui/BaseConfirmModal.vue'
import BaseButton from '@/components/ui/BaseButton.vue'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const toast = useToastStore()

// Export state
const exporting = ref(false)
const savingToServer = ref(false)

// Import state
const importing = ref(false)
const importFile = ref<File | null>(null)
const importFileName = ref('')
const importFileInput = ref<HTMLInputElement | null>(null)
const showImportConfirm = ref(false)

// Backup list state
const backups = ref<{ filename: string; size: number; modified_at: string }[]>([])
const loadingBackups = ref(false)

// Delete state
const showDeleteConfirm = ref(false)
const deletingFilename = ref('')
const deleting = ref(false)

// --- Export: Download ---
async function handleDownloadBackup() {
  exporting.value = true
  try {
    const res = await settingApi.exportDatabase()
    const blob = new Blob([res.data], { type: 'application/octet-stream' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')

    // Try to get filename from Content-Disposition header
    const disposition = res.headers['content-disposition']
    let filename = `cbt_patra_backup_${formatDateFile(new Date())}.dump`
    if (disposition) {
      const match = disposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
      if (match?.[1]) filename = match[1].replace(/['"]/g, '')
    }

    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success('Database berhasil diexport')
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal mengexport database')
  } finally {
    exporting.value = false
  }
}

// --- Export: Save to Server ---
async function handleSaveToServer() {
  savingToServer.value = true
  try {
    await settingApi.exportAndSave()
    toast.success('Backup disimpan di server')
    await fetchBackups()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menyimpan backup ke server')
  } finally {
    savingToServer.value = false
  }
}

// --- Import ---
function handleFileSelect(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) {
    importFile.value = file
    importFileName.value = file.name
  }
}

function askImport() {
  if (!importFile.value) {
    toast.error('Pilih file backup terlebih dahulu')
    return
  }
  showImportConfirm.value = true
}

async function handleImport() {
  showImportConfirm.value = false
  if (!importFile.value) return
  importing.value = true
  try {
    await settingApi.importDatabase(importFile.value)
    toast.success('Database berhasil diimport')
    importFile.value = null
    importFileName.value = ''
    if (importFileInput.value) importFileInput.value.value = ''
    await fetchBackups()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal mengimport database')
  } finally {
    importing.value = false
  }
}

// --- Backup List ---
async function fetchBackups() {
  loadingBackups.value = true
  try {
    const res = await settingApi.listDatabaseBackups()
    backups.value = res.data.data || []
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal memuat daftar backup')
  } finally {
    loadingBackups.value = false
  }
}

// --- Delete Backup ---
function askDelete(filename: string) {
  deletingFilename.value = filename
  showDeleteConfirm.value = true
}

async function handleDelete() {
  showDeleteConfirm.value = false
  deleting.value = true
  try {
    await settingApi.deleteDatabaseBackup(deletingFilename.value)
    toast.success('Backup berhasil dihapus')
    await fetchBackups()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? 'Gagal menghapus backup')
  } finally {
    deleting.value = false
    deletingFilename.value = ''
  }
}

// --- Helpers ---
function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

function formatDate(iso: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDateFile(d: Date): string {
  return d.toISOString().replace(/[:.]/g, '-').slice(0, 19)
}

onMounted(() => {
  fetchBackups()
})
</script>

<template>
  <BasePageHeader
    title="Manajemen Database"
    subtitle="Export, import, dan kelola backup database sistem"
    :breadcrumbs="[
      { label: 'Pengaturan', to: '/admin/settings' },
      { label: 'Manajemen Database' },
    ]"
  />

  <div class="row g-3">
    <!-- Export Database -->
    <div class="col-md-6">
      <div class="card h-100">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-database-export me-2"></i>Export Database
          </h3>
        </div>
        <div class="card-body d-flex flex-column gap-3">
          <p class="text-muted small mb-0">
            Export seluruh database ke file backup. Anda dapat mengunduh langsung atau menyimpan ke server.
          </p>
          <div class="d-flex flex-wrap gap-2">
            <BaseButton variant="primary" :loading="exporting" @click="handleDownloadBackup">
              <i class="ti ti-download me-1"></i>Download Backup
            </BaseButton>
            <BaseButton variant="ghost" :loading="savingToServer" @click="handleSaveToServer">
              <i class="ti ti-server me-1"></i>Simpan ke Server
            </BaseButton>
          </div>
        </div>
      </div>
    </div>

    <!-- Import Database -->
    <div class="col-md-6">
      <div class="card h-100 border-warning">
        <div class="card-header bg-warning-subtle">
          <h3 class="card-title">
            <i class="ti ti-database-import me-2 text-warning"></i>Import Database
          </h3>
        </div>
        <div class="card-body d-flex flex-column gap-3">
          <div class="alert alert-warning mb-0">
            <div class="d-flex align-items-center gap-2">
              <i class="ti ti-alert-triangle fs-3 text-warning"></i>
              <div class="small">
                <strong>Perhatian:</strong> Import akan <strong>MENIMPA</strong> seluruh data database yang ada. Pastikan Anda sudah membuat backup terlebih dahulu.
              </div>
            </div>
          </div>

          <div>
            <label class="form-label">Pilih File Backup</label>
            <input
              ref="importFileInput"
              type="file"
              accept=".dump,.sql,.sql.gz"
              class="form-control"
              :disabled="importing"
              @change="handleFileSelect"
            />
            <div v-if="importFileName" class="form-hint mt-1">
              <i class="ti ti-file me-1"></i>File dipilih: <strong>{{ importFileName }}</strong>
            </div>
          </div>

          <div>
            <BaseButton
              variant="danger"
              :loading="importing"
              :disabled="!importFile"
              @click="askImport"
            >
              <i class="ti ti-upload me-1"></i>Mulai Import
            </BaseButton>
          </div>

          <div v-if="importing" class="alert alert-info mb-0">
            <div class="d-flex align-items-center gap-2">
              <span class="spinner-border spinner-border-sm text-info"></span>
              <div class="small">
                Mengimport database... jangan tutup halaman ini.
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Riwayat Backup -->
    <div class="col-12">
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-history me-2"></i>Riwayat Backup di Server
          </h3>
          <div class="card-options">
            <button class="btn btn-sm btn-ghost-secondary" :disabled="loadingBackups" @click="fetchBackups">
              <span v-if="loadingBackups" class="spinner-border spinner-border-sm me-1"></span>
              <i v-else class="ti ti-refresh me-1"></i>Refresh
            </button>
          </div>
        </div>
        <div class="card-body p-0">
          <!-- Loading -->
          <div v-if="loadingBackups" class="p-4 text-center text-muted">
            <span class="spinner-border spinner-border-sm me-2"></span>Memuat daftar backup...
          </div>

          <!-- Empty -->
          <div v-else-if="backups.length === 0" class="p-4 text-center text-muted">
            <i class="ti ti-database-off d-block mb-2" style="font-size: 2rem;"></i>
            Belum ada file backup di server.
          </div>

          <!-- Table -->
          <div v-else class="table-responsive">
            <table class="table table-vcenter card-table">
              <thead>
                <tr>
                  <th>Nama File</th>
                  <th>Ukuran</th>
                  <th>Tanggal</th>
                  <th class="w-1"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="backup in backups" :key="backup.filename">
                  <td>
                    <div class="d-flex align-items-center gap-2">
                      <i class="ti ti-file-database text-primary"></i>
                      <span class="text-reset">{{ backup.filename }}</span>
                    </div>
                  </td>
                  <td class="text-muted">{{ formatBytes(backup.size) }}</td>
                  <td class="text-muted">{{ formatDate(backup.modified_at) }}</td>
                  <td>
                    <button
                      class="btn btn-ghost-danger btn-sm"
                      :disabled="deleting && deletingFilename === backup.filename"
                      @click="askDelete(backup.filename)"
                    >
                      <span
                        v-if="deleting && deletingFilename === backup.filename"
                        class="spinner-border spinner-border-sm"
                      ></span>
                      <i v-else class="ti ti-trash"></i>
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Import Confirmation Modal -->
  <BaseConfirmModal
    v-if="showImportConfirm"
    title="Konfirmasi Import Database"
    message="Import akan menimpa SELURUH data database. Pastikan Anda sudah membuat backup. Apakah Anda yakin ingin melanjutkan?"
    confirm-label="Ya, Import Sekarang"
    confirm-variant="danger"
    @confirm="handleImport"
    @close="showImportConfirm = false"
  />

  <!-- Delete Confirmation Modal -->
  <BaseConfirmModal
    v-if="showDeleteConfirm"
    title="Hapus Backup"
    :message="`Hapus file backup '${deletingFilename}'? Tindakan ini tidak dapat dibatalkan.`"
    confirm-label="Ya, Hapus"
    confirm-variant="danger"
    @confirm="handleDelete"
    @close="showDeleteConfirm = false"
  />
</template>
