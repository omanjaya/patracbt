<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import client from '../../api/client'
import { getAvatarUrl } from '../../utils/avatar'
import BaseModal from '../../components/ui/BaseModal.vue'
import BaseInput from '../../components/ui/BaseInput.vue'
import BaseButton from '../../components/ui/BaseButton.vue'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

interface RombelItem {
  id: number
  name: string
  grade_level?: string | null
}

interface RoomItem {
  id: number
  name: string
  capacity?: number
}

interface UserProfile {
  id: number
  name: string
  username: string
  email?: string
  role: string
  avatar_path?: string | null
  profile?: {
    nis?: string
    nip?: string
    class?: string
    major?: string
    phone?: string
    rombel_id?: number | null
    room_id?: number | null
  }
}

const user = ref<UserProfile | null>(null)
const loading = ref(true)
const saving = ref(false)
const savingPw = ref(false)
const successMsg = ref('')
const errorMsg = ref('')
const pwSuccess = ref('')
const pwError = ref('')

// Avatar state
const avatarInput = ref<HTMLInputElement | null>(null)
const uploadingAvatar = ref(false)
const avatarError = ref('')
const avatarPreview = ref<string | null>(null)

// Cropper State
const showCropModal = ref(false)
const cropImageSrc = ref('')
const cropperImgRef = ref<HTMLImageElement | null>(null)
let cropperInstance: Cropper | null = null

const form = ref({ name: '', email: '', nis: '', nip: '', class: '', major: '', phone: '' })
const pwForm = ref({ current_password: '', new_password: '', confirm_password: '' })
const profileSubmitted = ref(false)
const pwSubmitted = ref(false)

// Real-time validation errors
const profileErrors = reactive<Record<string, string>>({})
const pwErrors = reactive<Record<string, string>>({})

function validateProfileField(field: string) {
  delete profileErrors[field]
  if (field === 'name' && !form.value.name.trim()) {
    profileErrors.name = 'Nama lengkap wajib diisi'
  }
}

function validatePwField(field: string) {
  delete pwErrors[field]
  switch (field) {
    case 'current_password':
      if (!pwForm.value.current_password) pwErrors.current_password = 'Password lama wajib diisi'
      break
    case 'new_password':
      if (!pwForm.value.new_password) pwErrors.new_password = 'Password baru wajib diisi'
      else if (pwForm.value.new_password.length < 6) pwErrors.new_password = 'Password baru minimal 6 karakter'
      break
    case 'confirm_password':
      if (pwForm.value.confirm_password && pwForm.value.new_password !== pwForm.value.confirm_password) pwErrors.confirm_password = 'Konfirmasi password tidak sesuai'
      break
  }
}

// Rombel & Room for peserta display
const rombels = ref<RombelItem[]>([])
const rooms = ref<RoomItem[]>([])
const loadingRombels = ref(false)
const loadingRooms = ref(false)

const currentRombelName = computed(() => {
  if (!user.value?.profile?.rombel_id) return '–'
  const r = rombels.value.find(rb => rb.id === user.value!.profile!.rombel_id)
  return r ? r.name : '–'
})

const currentRoomName = computed(() => {
  if (!user.value?.profile?.room_id) return '–'
  const r = rooms.value.find(rm => rm.id === user.value!.profile!.room_id)
  return r ? r.name : '–'
})

const currentAvatarUrl = computed(() => {
  if (avatarPreview.value) return avatarPreview.value
  if (user.value?.avatar_path) return user.value.avatar_path
  return getAvatarUrl(user.value?.id ?? 0)
})

function triggerAvatarUpload() {
  avatarInput.value?.click()
}

async function handleAvatarUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  // Validate type
  const allowed = ['image/jpeg', 'image/png', 'image/webp']
  if (!allowed.includes(file.type)) {
    avatarError.value = 'Format tidak didukung. Gunakan JPEG, PNG, atau WebP.'
    return
  }
  // Validate size (2MB)
  if (file.size > 2 * 1024 * 1024) {
    avatarError.value = 'Ukuran file maksimal 2MB.'
    return
  }

  // Show preview immediately in cropper
  const reader = new FileReader()
  reader.onload = (e) => {
    cropImageSrc.value = e.target?.result as string
    showCropModal.value = true
    // init cropper shortly after modal mount
    setTimeout(() => {
      if (cropperImgRef.value) {
        if (cropperInstance) cropperInstance.destroy()
        cropperInstance = new Cropper(cropperImgRef.value, {
          aspectRatio: 1,
          viewMode: 1,
          autoCropArea: 1,
        })
      }
    }, 150)
  }
  reader.readAsDataURL(file)
}

function cancelCrop() {
  showCropModal.value = false
  if (cropperInstance) {
    cropperInstance.destroy()
    cropperInstance = null
  }
  if (avatarInput.value) avatarInput.value.value = ''
}

function confirmCrop() {
  if (!cropperInstance) return
  const canvas = cropperInstance.getCroppedCanvas({ width: 300, height: 300 })
  canvas.toBlob(async (blob) => {
    if (!blob) {
      avatarError.value = 'Gagal memotong gambar'
      return
    }
    
    // hide modal & show loading
    showCropModal.value = false
    uploadingAvatar.value = true
    avatarError.value = ''
    
    // set preview to cropped image
    avatarPreview.value = canvas.toDataURL()

    try {
      const formData = new FormData()
      formData.append('avatar', blob, 'avatar.jpg')
      const res = await client.post('/api/v1/profile/avatar', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      if (user.value) {
        user.value.avatar_path = res.data?.data?.avatar_path ?? null
      }
    } catch (e: any) {
      avatarError.value = e?.response?.data?.message ?? 'Gagal mengupload avatar'
      avatarPreview.value = null
    } finally {
      uploadingAvatar.value = false
      cancelCrop()
    }
  }, 'image/jpeg', 0.9)
}

async function loadProfile() {
  loading.value = true
  try {
    const res = await client.get('/api/v1/profile')
    user.value = res.data.data
    const u = user.value!
    form.value = {
      name: u.name ?? '',
      email: u.email ?? '',
      nis: u.profile?.nis ?? '',
      nip: u.profile?.nip ?? '',
      class: u.profile?.class ?? '',
      major: u.profile?.major ?? '',
      phone: u.profile?.phone ?? '',
    }
  } catch {
    errorMsg.value = 'Gagal memuat profil'
  } finally {
    loading.value = false
  }
}

async function saveProfile() {
  profileSubmitted.value = true
  successMsg.value = ''
  errorMsg.value = ''
  if (!form.value.name.trim()) {
    errorMsg.value = 'Nama lengkap wajib diisi'
    return
  }
  saving.value = true
  try {
    await client.put('/api/v1/profile', {
      name: form.value.name,
      email: form.value.email || undefined,
      nis: form.value.nis || undefined,
      nip: form.value.nip || undefined,
      class: form.value.class || undefined,
      major: form.value.major || undefined,
      phone: form.value.phone || undefined,
    })
    successMsg.value = 'Profil berhasil disimpan'
    await loadProfile()
  } catch (e: any) {
    errorMsg.value = e?.response?.data?.message ?? 'Gagal menyimpan profil'
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  pwSubmitted.value = true
  pwSuccess.value = ''
  pwError.value = ''
  if (!pwForm.value.current_password) {
    pwError.value = 'Password lama wajib diisi'
    return
  }
  if (!pwForm.value.new_password || pwForm.value.new_password.length < 6) {
    pwError.value = 'Password baru minimal 6 karakter'
    return
  }
  if (pwForm.value.new_password !== pwForm.value.confirm_password) {
    pwError.value = 'Konfirmasi password tidak sesuai'
    return
  }
  savingPw.value = true
  try {
    await client.put('/api/v1/profile/password', {
      current_password: pwForm.value.current_password,
      new_password: pwForm.value.new_password,
    })
    pwSuccess.value = 'Password berhasil diubah'
    pwForm.value = { current_password: '', new_password: '', confirm_password: '' }
  } catch (e: any) {
    pwError.value = e?.response?.data?.message ?? e?.response?.data?.error ?? 'Gagal mengubah password'
  } finally {
    savingPw.value = false
  }
}

async function loadRombelsAndRooms() {
  if (user.value?.role !== 'peserta') return
  loadingRombels.value = true
  loadingRooms.value = true
  try {
    const [rombelRes, roomRes] = await Promise.all([
      client.get('/admin/rombels', { params: { per_page: 200 } }),
      client.get('/admin/rooms', { params: { per_page: 200 } }),
    ])
    rombels.value = rombelRes.data.data ?? []
    rooms.value = roomRes.data.data ?? []
  } catch {
    // Peserta may not have access to admin endpoints; ignore
  } finally {
    loadingRombels.value = false
    loadingRooms.value = false
  }
}

onMounted(async () => {
  await loadProfile()
  await loadRombelsAndRooms()
})
</script>
<template>
  <div class="page-header d-print-none mb-3">
    <div class="row align-items-center">
      <div class="col">
        <h2 class="page-title">Profil Saya</h2>
        <p class="text-muted mb-0">Kelola informasi akun Anda</p>
      </div>
    </div>
  </div>

  <div v-if="loading" class="p-5 text-center text-muted">
    <span class="spinner-border spinner-border-sm me-2"></span>Memuat profil...
  </div>

  <div v-else class="row g-3">
    <!-- Profil Info -->
    <div class="col-lg-8">
      <div class="card">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-user me-2"></i>Informasi Profil
          </h3>
        </div>
        <div class="card-body">
          <div v-if="successMsg" class="alert alert-success mb-3">{{ successMsg }}</div>
          <div v-if="errorMsg" class="alert alert-danger mb-3">{{ errorMsg }}</div>

          <form id="profile-form" @submit.prevent="saveProfile">
            <div class="row g-3">
              <div class="col-md-6">
                <BaseInput
                  v-model="form.name"
                  label="Nama Lengkap *"
                  type="text"
                  :error="profileErrors.name || (profileSubmitted && !form.name.trim() ? 'Nama lengkap wajib diisi' : '')"
                  @blur="validateProfileField('name')"
                  @input="profileErrors.name = ''"
                />
              </div>
              <div class="col-md-6">
                <BaseInput v-model="form.email" label="Email" type="email" />
              </div>
              <div class="col-md-6">
                <BaseInput v-model="form.nis" label="NIS" type="text" />
              </div>
              <div class="col-md-6">
                <BaseInput v-model="form.nip" label="NIP" type="text" />
              </div>
              <div class="col-md-6">
                <BaseInput v-model="form.class" label="Kelas" type="text" />
              </div>
              <div class="col-md-6">
                <BaseInput v-model="form.major" label="Jurusan" type="text" />
              </div>
              <div class="col-12">
                <BaseInput v-model="form.phone" label="No. Telepon" type="text" />
              </div>
              <!-- Rombel & Room (read-only for peserta) -->
              <template v-if="user?.role === 'peserta'">
                <div class="col-md-6">
                  <label class="form-label">
                    <i class="ti ti-users-group me-1"></i>Rombel
                  </label>
                  <input
                    type="text"
                    class="form-control"
                    :value="currentRombelName"
                    disabled
                    readonly
                  />
                  <div class="form-hint text-muted small">Rombel diatur oleh admin</div>
                </div>
                <div class="col-md-6">
                  <label class="form-label">
                    <i class="ti ti-door me-1"></i>Ruang Ujian
                  </label>
                  <input
                    type="text"
                    class="form-control"
                    :value="currentRoomName"
                    disabled
                    readonly
                  />
                  <div class="form-hint text-muted small">Ruang diatur oleh admin</div>
                </div>
              </template>
            </div>
          </form>
        </div>
        <div class="card-footer d-flex align-items-center justify-content-between">
          <div class="d-flex align-items-center gap-2">
            <span class="badge bg-primary-lt text-primary">{{ user?.role }}</span>
            <span class="text-muted small">@{{ user?.username }}</span>
          </div>
          <BaseButton type="submit" variant="primary" :loading="saving" form="profile-form">
            <i class="ti ti-device-floppy me-1"></i>Simpan Profil
          </BaseButton>
        </div>
      </div>
    </div>

    <!-- Ubah Password -->
    <div class="col-lg-4">
      <!-- Avatar card -->
      <div class="card mb-3">
        <div class="card-body text-center py-4">
          <div
            class="avatar avatar-xl rounded-circle mb-3 d-block mx-auto position-relative"
            :style="`background-image:url(${currentAvatarUrl}); cursor:pointer;`"
            role="button"
            tabindex="0"
            aria-label="Ganti foto profil"
            title="Klik untuk ganti foto"
            @click="triggerAvatarUpload"
            @keydown.enter.prevent="triggerAvatarUpload"
            @keydown.space.prevent="triggerAvatarUpload"
          >
            <span
              v-if="uploadingAvatar"
              class="position-absolute top-50 start-50 translate-middle"
              style="background:rgba(0,0,0,0.5); border-radius:50%; width:100%; height:100%; display:flex; align-items:center; justify-content:center;"
            >
              <span class="spinner-border spinner-border-sm text-white"></span>
            </span>
          </div>
          <input
            ref="avatarInput"
            type="file"
            accept="image/jpeg,image/png,image/webp"
            class="d-none"
            @change="handleAvatarUpload"
          />
          <div class="mb-2">
            <a href="#" class="small text-primary" @click.prevent="triggerAvatarUpload">
              <i class="ti ti-camera me-1"></i>Ganti Foto
            </a>
          </div>
          <div v-if="avatarError" class="alert alert-danger py-1 px-2 small mb-2">{{ avatarError }}</div>
          <div class="fw-semibold">{{ user?.name }}</div>
          <div class="text-muted small">@{{ user?.username }}</div>
          <span class="badge bg-primary-lt text-primary mt-1">{{ user?.role }}</span>
        </div>
      </div>

      <div class="card">
        <div class="card-header">
          <h3 class="card-title">
            <i class="ti ti-lock me-2"></i>Ubah Password
          </h3>
        </div>
        <div class="card-body">
          <div v-if="pwSuccess" class="alert alert-success mb-3">{{ pwSuccess }}</div>
          <div v-if="pwError" class="alert alert-danger mb-3">{{ pwError }}</div>

          <form @submit.prevent="changePassword">
            <BaseInput
              v-model="pwForm.current_password"
              label="Password Lama *"
              type="password"
              :error="pwErrors.current_password || (pwSubmitted && !pwForm.current_password ? 'Password lama wajib diisi' : '')"
              @blur="validatePwField('current_password')"
              @input="pwErrors.current_password = ''"
            />
            <BaseInput
              v-model="pwForm.new_password"
              label="Password Baru *"
              type="password"
              hint="Minimal 6 karakter"
              :error="pwErrors.new_password || (pwSubmitted && pwForm.new_password.length > 0 && pwForm.new_password.length < 6 ? 'Password baru minimal 6 karakter' : pwSubmitted && !pwForm.new_password ? 'Password baru wajib diisi' : '')"
              @blur="validatePwField('new_password')"
              @input="pwErrors.new_password = ''"
            />
            <BaseInput
              v-model="pwForm.confirm_password"
              label="Konfirmasi Password Baru *"
              type="password"
              hint="Minimal 6 karakter"
              :error="pwErrors.confirm_password || (pwSubmitted && pwForm.confirm_password && pwForm.new_password !== pwForm.confirm_password ? 'Konfirmasi password tidak sesuai' : '')"
              @blur="validatePwField('confirm_password')"
              @input="pwErrors.confirm_password = ''"
            />
            <BaseButton type="submit" variant="primary" :loading="savingPw" class="w-100">
              <i class="ti ti-lock me-1"></i>Ubah Password
            </BaseButton>
          </form>
        </div>
      </div>
    </div>
  </div>

  <!-- Crop Modal -->
  <BaseModal v-if="showCropModal" title="Potong Foto Profil" size="md" @close="cancelCrop">
    <!-- Body -->
    <div style="max-height: 400px; text-align: center;">
      <img ref="cropperImgRef" :src="cropImageSrc" style="max-width: 100%; display: block;" />
    </div>

    <!-- Footer -->
    <template #footer>
      <BaseButton variant="secondary" @click="cancelCrop">Batal</BaseButton>
      <BaseButton variant="primary" @click="confirmCrop">
        <i class="ti ti-check me-2"></i>Simpan Potongan
      </BaseButton>
    </template>
  </BaseModal>
</template>
