<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import {
  examApi,
  type ExamSchedule,
  type CreateSchedulePayload,
} from '../../../api/exam.api'
import { questionBankApi, type QuestionBank } from '../../../api/question_bank.api'
import { rombelApi, type Rombel } from '../../../api/rombel.api'
import { tagApi, type Tag as TagType } from '../../../api/tag.api'
import { userApi, type UserItem } from '../../../api/user.api'
import BasePageHeader from '@/components/ui/BasePageHeader.vue'

const route = useRoute()
const router = useRouter()

const scheduleId = computed(() => route.params.id ? Number(route.params.id) : null)
const isEdit = computed(() => scheduleId.value !== null)

// Unsaved changes guard
const formDirty = ref(false)
const formLoaded = ref(false)

function markDirty() {
  if (formLoaded.value) formDirty.value = true
}

function beforeUnloadHandler(e: BeforeUnloadEvent) {
  if (formDirty.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}

onMounted(() => {
  window.addEventListener('beforeunload', beforeUnloadHandler)
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', beforeUnloadHandler)
})

onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
})

onBeforeRouteLeave((_to, _from, next) => {
  if (formDirty.value) {
    const answer = window.confirm('Anda memiliki perubahan yang belum disimpan. Yakin ingin meninggalkan halaman ini?')
    next(answer)
  } else {
    next()
  }
})

const loading = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const isEditable = ref(true)

const allBanks = ref<QuestionBank[]>([])
const allRombels = ref<Rombel[]>([])
const allTags = ref<TagType[]>([])
const allSchedules = ref<ExamSchedule[]>([])

interface BankRow { question_bank_id: number; question_count: number; weight: number }
interface UserOption { id: number; name: string; username: string }
const form = reactive({
  name: '',
  start_time: '',
  end_time: '',
  duration_minutes: 60,
  allow_see_result: true,
  max_violations: 3,
  randomize_questions: false,
  randomize_options: false,
  next_exam_schedule_id: null as number | null,
  late_policy: 'allow_full_time' as string,
  min_working_time: 0,
  detect_cheating: true,
  cheating_limit: 0,
  show_score_after: 'immediately' as string,
  selected_banks: [] as BankRow[],
  rombel_ids: [] as number[],
  tag_ids: [] as number[],
  include_users: [] as UserOption[],
  exclude_users: [] as UserOption[],
})

// User search state
const showIncludeSection = ref(false)
const showExcludeSection = ref(false)
const includeSearchQuery = ref('')
const excludeSearchQuery = ref('')
const userSearchResults = ref<UserItem[]>([])
const searchingUsers = ref(false)
let searchTimeout: ReturnType<typeof setTimeout> | null = null

async function searchUsers(query: string) {
  if (query.length < 2) {
    userSearchResults.value = []
    return
  }
  searchingUsers.value = true
  try {
    const res = await userApi.list({ search: query, per_page: 20, role: 'peserta' })
    userSearchResults.value = res.data.data ?? []
  } finally {
    searchingUsers.value = false
  }
}

function onIncludeSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => searchUsers(includeSearchQuery.value), 300)
}

function onExcludeSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => searchUsers(excludeSearchQuery.value), 300)
}

function addIncludeUser(user: UserItem) {
  if (!form.include_users.find(u => u.id === user.id)) {
    form.include_users.push({ id: user.id, name: user.name, username: user.username })
  }
  includeSearchQuery.value = ''
  userSearchResults.value = []
}

function removeIncludeUser(userId: number) {
  form.include_users = form.include_users.filter(u => u.id !== userId)
}

function addExcludeUser(user: UserItem) {
  if (!form.exclude_users.find(u => u.id === user.id)) {
    form.exclude_users.push({ id: user.id, name: user.name, username: user.username })
  }
  excludeSearchQuery.value = ''
  userSearchResults.value = []
}

function removeExcludeUser(userId: number) {
  form.exclude_users = form.exclude_users.filter(u => u.id !== userId)
}

async function fetchDependencies() {
  const [banksRes, rombelsRes, tagsRes, schedulesRes] = await Promise.all([
    questionBankApi.list({ per_page: 200 }),
    rombelApi.list({ per_page: 200 }),
    tagApi.listAll(),
    examApi.listSchedules({ per_page: 200 }),
  ])
  allBanks.value = banksRes.data.data ?? []
  allRombels.value = rombelsRes.data.data ?? []
  allTags.value = tagsRes.data.data ?? []
  allSchedules.value = schedulesRes.data.data ?? []
}

async function fetchSchedule(id: number) {
  loading.value = true
  try {
    const res = await examApi.getSchedule(id)
    const s: ExamSchedule = res.data.data
    form.name = s.name
    form.start_time = s.start_time.slice(0, 16)
    form.end_time = s.end_time.slice(0, 16)
    form.duration_minutes = s.duration_minutes
    form.allow_see_result = s.allow_see_result
    form.max_violations = s.max_violations
    form.randomize_questions = s.randomize_questions
    form.randomize_options = s.randomize_options
    form.next_exam_schedule_id = s.next_exam_schedule_id ?? null
    form.late_policy = s.late_policy ?? 'allow_full_time'
    form.min_working_time = s.min_working_time ?? 0
    form.detect_cheating = s.detect_cheating ?? true
    form.cheating_limit = s.cheating_limit ?? 0
    form.show_score_after = s.show_score_after ?? 'immediately'
    form.selected_banks = (s.question_banks ?? []).map(b => ({
      question_bank_id: b.question_bank_id,
      question_count: b.question_count,
      weight: b.weight ?? 1,
    }))
    form.rombel_ids = (s.rombels ?? []).map(r => r.rombel_id)
    form.tag_ids = (s.tags ?? []).map(t => t.tag_id)
    // Load include/exclude users
    const includeUsers = (s.users ?? []).filter(u => u.type === 'include')
    const excludeUsers = (s.users ?? []).filter(u => u.type === 'exclude')
    form.include_users = includeUsers.map(u => ({
      id: u.user_id,
      name: u.user?.name ?? `User #${u.user_id}`,
      username: u.user?.username ?? '',
    }))
    form.exclude_users = excludeUsers.map(u => ({
      id: u.user_id,
      name: u.user?.name ?? `User #${u.user_id}`,
      username: u.user?.username ?? '',
    }))
    if (form.include_users.length > 0) showIncludeSection.value = true
    if (form.exclude_users.length > 0) showExcludeSection.value = true
    // If status is active or finished, form is partially locked
    if (s.status === 'active' || s.status === 'finished') {
      isEditable.value = false
    }
  } finally {
    loading.value = false
  }
}

function addBank() {
  form.selected_banks.push({ question_bank_id: 0, question_count: 0, weight: 1 })
}

function removeBank(i: number) {
  form.selected_banks.splice(i, 1)
}

// Validation
const formErrors = reactive<Record<string, string>>({})

function validateField(field: string) {
  delete formErrors[field]
  switch (field) {
    case 'name':
      if (!form.name.trim()) formErrors.name = 'Nama ujian wajib diisi'
      break
    case 'start_time':
      if (!form.start_time) formErrors.start_time = 'Waktu mulai wajib diisi'
      else if (!isEdit.value && new Date(form.start_time) <= new Date()) formErrors.start_time = 'Waktu mulai harus di masa depan'
      break
    case 'end_time':
      if (!form.end_time) formErrors.end_time = 'Waktu selesai wajib diisi'
      else if (form.start_time && new Date(form.end_time) <= new Date(form.start_time)) formErrors.end_time = 'Waktu selesai harus setelah waktu mulai'
      break
    case 'duration_minutes':
      if (!form.duration_minutes || form.duration_minutes <= 0) formErrors.duration_minutes = 'Durasi harus lebih dari 0'
      break
  }
}

function validateSchedule(): boolean {
  Object.keys(formErrors).forEach(k => delete formErrors[k])

  if (!form.name.trim()) formErrors.name = 'Nama ujian wajib diisi'

  if (!form.start_time) {
    formErrors.start_time = 'Waktu mulai wajib diisi'
  } else if (!isEdit.value && new Date(form.start_time) <= new Date()) {
    formErrors.start_time = 'Waktu mulai harus di masa depan'
  }

  if (!form.end_time) {
    formErrors.end_time = 'Waktu selesai wajib diisi'
  } else if (form.start_time && new Date(form.end_time) <= new Date(form.start_time)) {
    formErrors.end_time = 'Waktu selesai harus setelah waktu mulai'
  }

  if (!form.duration_minutes || form.duration_minutes <= 0) {
    formErrors.duration_minutes = 'Durasi harus lebih dari 0'
  }

  const validBanks = form.selected_banks.filter(b => b.question_bank_id > 0)
  if (validBanks.length === 0) {
    formErrors.banks = 'Minimal 1 bank soal harus ditambahkan'
  }

  const hasTarget = form.rombel_ids.length > 0 || form.tag_ids.length > 0 || form.include_users.length > 0
  if (!hasTarget) {
    formErrors.target = 'Minimal 1 target peserta (rombel, tag, atau peserta khusus) harus dipilih'
  }

  return Object.keys(formErrors).length === 0
}

async function handleSubmit() {
  if (!validateSchedule()) {
    errorMsg.value = 'Mohon perbaiki kesalahan pada form sebelum menyimpan.'
    return
  }
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const payload: CreateSchedulePayload = {
      name: form.name,
      start_time: new Date(form.start_time).toISOString(),
      end_time: new Date(form.end_time).toISOString(),
      duration_minutes: form.duration_minutes,
      allow_see_result: form.allow_see_result,
      max_violations: form.max_violations,
      randomize_questions: form.randomize_questions,
      randomize_options: form.randomize_options,
      next_exam_schedule_id: form.next_exam_schedule_id ?? undefined,
      late_policy: form.late_policy,
      min_working_time: form.min_working_time,
      detect_cheating: form.detect_cheating,
      cheating_limit: form.cheating_limit,
      show_score_after: form.show_score_after,
      question_banks: form.selected_banks.filter(b => b.question_bank_id > 0).map(b => ({
        question_bank_id: b.question_bank_id,
        question_count: b.question_count,
        weight: b.weight || 1,
      })),
      rombel_ids: form.rombel_ids,
      tag_ids: form.tag_ids,
      include_users: form.include_users.map(u => u.id),
      exclude_users: form.exclude_users.map(u => u.id),
    }
    if (isEdit.value && scheduleId.value) {
      await examApi.updateSchedule(scheduleId.value, payload)
      formDirty.value = false
      successMsg.value = 'Jadwal ujian berhasil diperbarui.'
    } else {
      formDirty.value = false
      await examApi.createSchedule(payload)
      router.push('/admin/exam-schedules')
    }
  } catch (err: unknown) {
    const e = err as { response?: { data?: { message?: string } } }
    errorMsg.value = e?.response?.data?.message ?? 'Gagal menyimpan. Periksa kembali data.'
  } finally {
    saving.value = false
  }
}

watch(form, () => markDirty(), { deep: true })

onMounted(async () => {
  await fetchDependencies()
  if (isEdit.value && scheduleId.value) {
    await fetchSchedule(scheduleId.value)
  }
  // Mark form as loaded after Vue finishes processing reactive updates so watchers don't trigger dirty flag
  await nextTick()
  formLoaded.value = true
})
</script>

<template>
  <div>
    <!-- Page Header -->
    <BasePageHeader
      :title="isEdit ? 'Edit Jadwal Ujian' : 'Buat Jadwal Ujian Baru'"
      :subtitle="isEdit ? 'Perbarui pengaturan jadwal ujian CBT' : 'Buat jadwal ujian CBT baru'"
      :breadcrumbs="[{ label: 'Jadwal Ujian', to: '/admin/exam-schedules' }, { label: isEdit ? 'Edit' : 'Buat Baru' }]"
    >
      <template #actions>
        <router-link v-if="isEdit && scheduleId" :to="`${route.path.replace('/edit', '/preview')}`" class="btn btn-outline-cyan">
          <i class="ti ti-eye me-1"></i>
          Preview Soal
        </router-link>
        <button class="btn btn-outline-secondary" @click="router.push('/admin/exam-schedules')">
          <i class="ti ti-arrow-bar-left me-1"></i>
          Kembali ke Daftar Jadwal
        </button>
      </template>
    </BasePageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border text-primary" role="status"></div>
      <p class="text-muted mt-2">Memuat data...</p>
    </div>

    <template v-else>
      <!-- Alert: not editable warning -->
      <div v-if="isEdit && !isEditable" class="alert alert-warning mb-3">
        <div class="d-flex align-items-start gap-2">
          <i class="ti ti-info-circle fs-4 flex-shrink-0 mt-1"></i>
          <div>
            <h4 class="alert-title">Jadwal ujian ini sudah <strong>DIMULAI</strong></h4>
            <div class="text-muted mt-1">
              <strong>Yang dapat diubah:</strong> Bobot nilai, opsi tampilkan hasil, waktu berakhir.<br>
              <strong>Yang TIDAK dapat diubah:</strong> Bank soal, durasi, waktu mulai, acak soal/opsi.
            </div>
          </div>
        </div>
      </div>

      <!-- Success alert -->
      <div v-if="successMsg" class="alert alert-success alert-dismissible mb-3">
        <div class="d-flex align-items-center gap-2">
          <i class="ti ti-circle-check"></i>
          <div>{{ successMsg }}</div>
        </div>
        <button type="button" class="btn-close" @click="successMsg = ''"></button>
      </div>

      <!-- Error alert -->
      <div v-if="errorMsg" class="alert alert-danger alert-dismissible mb-3">
        <div class="d-flex align-items-center gap-2">
          <i class="ti ti-alert-circle"></i>
          <div>{{ errorMsg }}</div>
        </div>
        <button type="button" class="btn-close" @click="errorMsg = ''"></button>
      </div>

      <form @submit.prevent="handleSubmit">
        <fieldset :disabled="saving">
        <div class="row g-4">
          <!-- Left: Main Settings -->
          <div class="col-lg-8">
            <div class="card mb-4">
              <div class="card-header">
                <h3 class="card-title">Pengaturan Ujian</h3>
              </div>
              <div class="card-body">
                <div class="row g-3">
                  <!-- Nama Ujian -->
                  <div class="col-12">
                    <label class="form-label">Nama Ujian <span class="text-danger">*</span></label>
                    <input
                      v-model="form.name"
                      type="text"
                      class="form-control"
                      :class="{ 'is-invalid': formErrors.name }"
                      placeholder="Contoh: UTS Matematika XII IPA"
                      @blur="validateField('name')"
                      @input="formErrors.name = ''"
                    />
                    <div v-if="formErrors.name" class="invalid-feedback">{{ formErrors.name }}</div>
                  </div>

                  <!-- Waktu Mulai -->
                  <div class="col-md-6">
                    <label class="form-label">Waktu Mulai <span class="text-danger">*</span></label>
                    <input
                      v-model="form.start_time"
                      type="datetime-local"
                      class="form-control"
                      :class="{ 'is-invalid': formErrors.start_time }"
                      :disabled="isEdit && !isEditable"
                      @blur="validateField('start_time')"
                      @input="formErrors.start_time = ''"
                    />
                    <div v-if="formErrors.start_time" class="invalid-feedback">{{ formErrors.start_time }}</div>
                  </div>

                  <!-- Waktu Selesai -->
                  <div class="col-md-6">
                    <label class="form-label">Waktu Selesai <span class="text-danger">*</span></label>
                    <input
                      v-model="form.end_time"
                      type="datetime-local"
                      class="form-control"
                      :class="{ 'is-invalid': formErrors.end_time }"
                      @blur="validateField('end_time')"
                      @input="formErrors.end_time = ''"
                    />
                    <div v-if="formErrors.end_time" class="invalid-feedback">{{ formErrors.end_time }}</div>
                  </div>

                  <!-- Durasi -->
                  <div class="col-md-6">
                    <label class="form-label">Durasi Pengerjaan (menit) <span class="text-danger">*</span></label>
                    <input
                      v-model.number="form.duration_minutes"
                      type="number"
                      min="1"
                      class="form-control"
                      :class="{ 'is-invalid': formErrors.duration_minutes }"
                      :disabled="isEdit && !isEditable"
                      @blur="validateField('duration_minutes')"
                      @input="formErrors.duration_minutes = ''"
                    />
                    <div v-if="formErrors.duration_minutes" class="invalid-feedback">{{ formErrors.duration_minutes }}</div>
                  </div>

                  <!-- Maks Pelanggaran -->
                  <div class="col-md-6">
                    <label class="form-label">Maks. Pelanggaran</label>
                    <input
                      v-model.number="form.max_violations"
                      type="number"
                      min="1"
                      class="form-control"
                    />
                    <small class="form-text text-muted">Sesi peserta akan dihentikan paksa setelah melebihi batas ini.</small>
                  </div>

                  <!-- Options -->
                  <div class="col-12">
                    <label class="form-label">Opsi Tambahan</label>
                    <div class="d-flex flex-wrap gap-3">
                      <label class="form-check mb-0">
                        <input type="checkbox" class="form-check-input" v-model="form.allow_see_result" />
                        <span class="form-check-label">Peserta bisa lihat hasil</span>
                      </label>
                      <label class="form-check mb-0">
                        <input
                          type="checkbox"
                          class="form-check-input"
                          v-model="form.randomize_questions"
                          :disabled="isEdit && !isEditable"
                        />
                        <span class="form-check-label">Acak urutan soal</span>
                      </label>
                      <label class="form-check mb-0">
                        <input
                          type="checkbox"
                          class="form-check-input"
                          v-model="form.randomize_options"
                          :disabled="isEdit && !isEditable"
                        />
                        <span class="form-check-label">Acak pilihan jawaban</span>
                      </label>
                    </div>
                  </div>

                  <!-- Multi-stage -->
                  <div class="col-12">
                    <label class="form-label">Bagian Berikutnya (Multi-Tahap)</label>
                    <select v-model="form.next_exam_schedule_id" class="form-select">
                      <option :value="null">— Tidak ada (ujian tunggal) —</option>
                      <option
                        v-for="s in allSchedules.filter(s => s.id !== scheduleId)"
                        :key="s.id"
                        :value="s.id"
                      >
                        {{ s.name }}
                      </option>
                    </select>
                    <small class="form-text text-muted">Isi jika ujian ini dilanjutkan ke sesi berikutnya otomatis setelah selesai.</small>
                  </div>
                </div>
              </div>
            </div>

            <!-- Bank Soal -->
            <div class="card mb-4">
              <div class="card-header">
                <h3 class="card-title"><i class="ti ti-books me-2"></i>Bank Soal</h3>
              </div>
              <div class="card-body">
                <div
                  v-for="(bank, i) in form.selected_banks"
                  :key="i"
                  class="d-flex gap-2 align-items-center mb-2"
                >
                  <select
                    v-model="bank.question_bank_id"
                    class="form-select"
                    :disabled="isEdit && !isEditable"
                  >
                    <option :value="0">Pilih bank soal</option>
                    <option v-for="b in allBanks" :key="b.id" :value="b.id">
                      {{ b.name }} ({{ b.question_count }} soal)
                    </option>
                  </select>
                  <input
                    v-model.number="bank.question_count"
                    type="number"
                    min="0"
                    class="form-control flex-shrink-0"
                    style="width: 110px"
                    placeholder="0=semua"
                    :disabled="isEdit && !isEditable"
                  />
                  <div class="input-group flex-shrink-0" style="width: 120px">
                    <span class="input-group-text" title="Bobot">
                      <i class="ti ti-scale"></i>
                    </span>
                    <input
                      v-model.number="bank.weight"
                      type="number"
                      min="0.1"
                      step="0.1"
                      class="form-control"
                      placeholder="Bobot"
                      title="Bobot nilai bank soal"
                    />
                  </div>
                  <button
                    type="button"
                    class="btn btn-sm btn-ghost-danger flex-shrink-0"
                    :disabled="isEdit && !isEditable"
                    @click="removeBank(i)"
                  >
                    <i class="ti ti-x"></i>
                  </button>
                </div>
                <div v-if="form.selected_banks.length === 0" class="text-muted small mb-2">
                  Belum ada bank soal ditambahkan.
                </div>
                <div v-if="formErrors.banks" class="text-danger small mb-2">{{ formErrors.banks }}</div>
                <button
                  type="button"
                  class="btn btn-sm btn-outline-secondary mt-1"
                  :disabled="isEdit && !isEditable"
                  @click="addBank"
                >
                  <i class="ti ti-plus me-1"></i>Tambah Bank Soal
                </button>
              </div>
            </div>

            <!-- Pengaturan Lanjutan -->
            <div class="card mb-4">
              <div class="card-header">
                <h3 class="card-title"><i class="ti ti-settings me-2"></i>Pengaturan Lanjutan</h3>
              </div>
              <div class="card-body">
                <div class="row g-3">
                  <!-- Kebijakan Keterlambatan -->
                  <div class="col-md-6">
                    <label class="form-label">Kebijakan Keterlambatan</label>
                    <select v-model="form.late_policy" class="form-select">
                      <option value="allow_full_time">Izinkan Waktu Penuh</option>
                      <option value="deduct_time">Kurangi Waktu</option>
                    </select>
                  </div>

                  <!-- Waktu Pengerjaan Minimum -->
                  <div class="col-md-6">
                    <label class="form-label">Waktu Pengerjaan Minimum (menit)</label>
                    <input
                      v-model.number="form.min_working_time"
                      type="number"
                      min="0"
                      class="form-control"
                    />
                    <small class="form-text text-muted">0 = tidak ada minimum. Peserta tidak bisa menyelesaikan ujian sebelum waktu ini.</small>
                  </div>

                  <!-- Deteksi Kecurangan -->
                  <div class="col-md-6">
                    <label class="form-label">Deteksi Kecurangan</label>
                    <div>
                      <label class="form-check form-switch mb-0">
                        <input type="checkbox" class="form-check-input" v-model="form.detect_cheating" />
                        <span class="form-check-label">{{ form.detect_cheating ? 'Aktif' : 'Nonaktif' }}</span>
                      </label>
                    </div>
                  </div>

                  <!-- Batas Pelanggaran -->
                  <div class="col-md-6" v-if="form.detect_cheating">
                    <label class="form-label">Batas Pelanggaran</label>
                    <input
                      v-model.number="form.cheating_limit"
                      type="number"
                      min="0"
                      class="form-control"
                    />
                    <small class="form-text text-muted">0 = tidak dibatasi. Jumlah pelanggaran sebelum sesi dihentikan otomatis.</small>
                  </div>

                  <!-- Tampilkan Nilai -->
                  <div class="col-md-6">
                    <label class="form-label">Tampilkan Nilai</label>
                    <select v-model="form.show_score_after" class="form-select">
                      <option value="immediately">Langsung Setelah Selesai</option>
                      <option value="after_end_time">Setelah Waktu Berakhir</option>
                      <option value="manual">Manual (Admin)</option>
                    </select>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Right: Participants -->
          <div class="col-lg-4">
            <!-- Rombel -->
            <div class="card mb-4">
              <div class="card-header">
                <h3 class="card-title"><i class="ti ti-users me-2"></i>Peserta (Rombel)</h3>
              </div>
              <div class="card-body">
                <small class="text-muted d-block mb-2">Kosongkan untuk semua rombel.</small>
                <div class="d-flex flex-column gap-1">
                  <label
                    v-for="r in allRombels"
                    :key="r.id"
                    class="form-check"
                  >
                    <input
                      type="checkbox"
                      class="form-check-input"
                      :value="r.id"
                      v-model="form.rombel_ids"
                    />
                    <span class="form-check-label">{{ r.name }}</span>
                    <span v-if="r.grade_level" class="text-muted small ms-1">({{ r.grade_level }})</span>
                  </label>
                </div>
                <div v-if="allRombels.length === 0" class="text-muted small">Belum ada rombel.</div>
                <div v-if="formErrors.target" class="text-danger small mt-2">{{ formErrors.target }}</div>
              </div>
            </div>

            <!-- Tags -->
            <div class="card mb-4" v-if="allTags.length > 0">
              <div class="card-header">
                <h3 class="card-title"><i class="ti ti-tag me-2"></i>Peserta (Tag)</h3>
              </div>
              <div class="card-body">
                <small class="text-muted d-block mb-2">Kosongkan untuk semua tag.</small>
                <div class="d-flex flex-column gap-1">
                  <label
                    v-for="t in allTags"
                    :key="t.id"
                    class="form-check"
                  >
                    <input
                      type="checkbox"
                      class="form-check-input"
                      :value="t.id"
                      v-model="form.tag_ids"
                    />
                    <span class="form-check-label d-flex align-items-center gap-1">
                      <span
                        class="d-inline-block rounded-circle flex-shrink-0"
                        style="width: 10px; height: 10px"
                        :style="{ background: t.color || 'var(--tblr-primary)' }"
                      ></span>
                      {{ t.name }}
                    </span>
                  </label>
                </div>
              </div>
            </div>

            <!-- Peserta Khusus (Include) -->
            <div class="card mb-4">
              <div class="card-header d-flex align-items-center justify-content-between">
                <h3 class="card-title mb-0"><i class="ti ti-user-check me-2"></i>Peserta Khusus (Include)</h3>
                <button type="button" class="btn btn-sm btn-ghost-primary" @click="showIncludeSection = !showIncludeSection">
                  <i :class="showIncludeSection ? 'ti ti-chevron-up' : 'ti ti-chevron-down'"></i>
                </button>
              </div>
              <div v-if="showIncludeSection" class="card-body">
                <small class="text-muted d-block mb-2">Jika diisi, HANYA peserta ini yang bisa mengakses ujian (selain filter rombel/tag).</small>
                <div class="position-relative mb-2">
                  <input
                    v-model="includeSearchQuery"
                    type="text"
                    class="form-control"
                    placeholder="Cari peserta..."
                    @input="onIncludeSearch"
                  />
                  <div
                    v-if="includeSearchQuery.length >= 2 && userSearchResults.length > 0"
                    class="dropdown-menu show w-100"
                    style="position: absolute; z-index: 1050; max-height: 200px; overflow-y: auto"
                  >
                    <a
                      v-for="user in userSearchResults"
                      :key="user.id"
                      class="dropdown-item cursor-pointer"
                      @click.prevent="addIncludeUser(user)"
                    >
                      <span class="fw-medium">{{ user.name }}</span>
                      <span class="text-muted ms-1">({{ user.username }})</span>
                    </a>
                  </div>
                  <div v-if="searchingUsers" class="text-muted small mt-1">Mencari...</div>
                </div>
                <div v-if="form.include_users.length > 0" class="d-flex flex-wrap gap-1">
                  <span
                    v-for="user in form.include_users"
                    :key="user.id"
                    class="badge bg-green-lt d-inline-flex align-items-center gap-1"
                  >
                    {{ user.name }} ({{ user.username }})
                    <a href="#" class="text-reset" @click.prevent="removeIncludeUser(user.id)">
                      <i class="ti ti-x" style="font-size: 12px"></i>
                    </a>
                  </span>
                </div>
                <div v-else class="text-muted small">Belum ada peserta khusus.</div>
              </div>
            </div>

            <!-- Blokir Peserta (Exclude) -->
            <div class="card mb-4">
              <div class="card-header d-flex align-items-center justify-content-between">
                <h3 class="card-title mb-0"><i class="ti ti-user-x me-2"></i>Blokir Peserta (Exclude)</h3>
                <button type="button" class="btn btn-sm btn-ghost-danger" @click="showExcludeSection = !showExcludeSection">
                  <i :class="showExcludeSection ? 'ti ti-chevron-up' : 'ti ti-chevron-down'"></i>
                </button>
              </div>
              <div v-if="showExcludeSection" class="card-body">
                <small class="text-muted d-block mb-2">Peserta ini TIDAK bisa mengakses ujian meskipun masuk rombel/tag yang dipilih.</small>
                <div class="position-relative mb-2">
                  <input
                    v-model="excludeSearchQuery"
                    type="text"
                    class="form-control"
                    placeholder="Cari peserta..."
                    @input="onExcludeSearch"
                  />
                  <div
                    v-if="excludeSearchQuery.length >= 2 && userSearchResults.length > 0"
                    class="dropdown-menu show w-100"
                    style="position: absolute; z-index: 1050; max-height: 200px; overflow-y: auto"
                  >
                    <a
                      v-for="user in userSearchResults"
                      :key="user.id"
                      class="dropdown-item cursor-pointer"
                      @click.prevent="addExcludeUser(user)"
                    >
                      <span class="fw-medium">{{ user.name }}</span>
                      <span class="text-muted ms-1">({{ user.username }})</span>
                    </a>
                  </div>
                  <div v-if="searchingUsers" class="text-muted small mt-1">Mencari...</div>
                </div>
                <div v-if="form.exclude_users.length > 0" class="d-flex flex-wrap gap-1">
                  <span
                    v-for="user in form.exclude_users"
                    :key="user.id"
                    class="badge bg-red-lt d-inline-flex align-items-center gap-1"
                  >
                    {{ user.name }} ({{ user.username }})
                    <a href="#" class="text-reset" @click.prevent="removeExcludeUser(user.id)">
                      <i class="ti ti-x" style="font-size: 12px"></i>
                    </a>
                  </span>
                </div>
                <div v-else class="text-muted small">Tidak ada peserta yang diblokir.</div>
              </div>
            </div>

            <!-- Submit Card -->
            <div class="card">
              <div class="card-body">
                <div class="d-grid gap-2">
                  <button type="submit" class="btn btn-primary" :disabled="saving">
                    <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>
                    <i v-else class="ti ti-device-floppy me-1"></i>
                    {{ isEdit ? 'Simpan Perubahan' : 'Buat Jadwal' }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-outline-secondary"
                    @click="router.push('/admin/exam-schedules')"
                  >
                    Batal
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
        </fieldset>
      </form>
    </template>
  </div>
</template>
