# Shared Components - CBT Patra (Vue.js)

**Tanggal:** 2026-03-05

---

## Komponen UI Base

### BaseButton.vue
```
Props:
  - variant: 'primary' | 'secondary' | 'danger' | 'ghost' | 'outline'
  - size: 'sm' | 'md' | 'lg'
  - loading: boolean
  - disabled: boolean
  - icon: string (optional)

Slot: default (label)

Usage:
<BaseButton variant="primary" :loading="isSubmitting" @click="submit">
    Simpan Jadwal
</BaseButton>
```

### BaseInput.vue
```
Props:
  - modelValue: string | number
  - label: string
  - placeholder: string
  - error: string
  - type: string (default: 'text')
  - required: boolean

Emits: update:modelValue

Usage:
<BaseInput v-model="form.name" label="Nama Jadwal" :error="errors.name" required />
```

### BaseModal.vue
```
Props:
  - modelValue: boolean (open/close)
  - title: string
  - size: 'sm' | 'md' | 'lg' | 'xl' | 'full'
  - closable: boolean (default: true)

Slots: default (body), footer

Usage:
<BaseModal v-model="showCreateModal" title="Buat Jadwal Ujian" size="lg">
    <form @submit="handleSubmit">...</form>
    <template #footer>
        <BaseButton @click="showCreateModal = false">Batal</BaseButton>
        <BaseButton variant="primary" type="submit">Simpan</BaseButton>
    </template>
</BaseModal>
```

### BaseTable.vue
```
Props:
  - columns: TableColumn[]
  - data: unknown[]
  - loading: boolean
  - pagination: PaginationMeta

Events: page-change, sort-change

Usage:
<BaseTable :columns="columns" :data="users" :loading="isLoading" :pagination="meta" />
```

### BaseAlert.vue
```
Props:
  - type: 'success' | 'warning' | 'danger' | 'info'
  - message: string
  - dismissible: boolean

Usage:
<BaseAlert type="warning" message="Waktu ujian kurang dari 5 menit!" />
```

### BaseBadge.vue
```
Props:
  - variant: 'upcoming' | 'active' | 'finished' | 'ongoing' | 'completed' | 'terminated'
  - pulse: boolean

Usage:
<BaseBadge variant="active" :pulse="true" />
```

### BaseAvatar.vue
```
Props:
  - name: string (untuk initials fallback)
  - src: string (optional)
  - size: 'xs' | 'sm' | 'md' | 'lg'

Usage:
<BaseAvatar name="Budi Santoso" :src="user.avatarUrl" size="md" />
```

### BasePagination.vue
```
Props:
  - currentPage: number
  - totalPages: number
  - perPage: number
  - total: number

Emits: page-change

Usage:
<BasePagination :current-page="page" :total-pages="meta.totalPages" @page-change="fetchData" />
```

### BaseDropdown.vue
```
Props:
  - items: DropdownItem[]
  - align: 'left' | 'right'

Slot: trigger

Usage:
<BaseDropdown :items="actionItems" align="right">
    <template #trigger>
        <BaseButton variant="ghost" icon="dots-vertical">Aksi</BaseButton>
    </template>
</BaseDropdown>
```

### BaseToast.vue (Global)
```
State di Pinia: useToastStore()

API:
  - toast.success('Data berhasil disimpan')
  - toast.error('Gagal memuat data')
  - toast.warning('Pelanggaran terdeteksi!')
  - toast.info('Ujian akan dimulai dalam 5 menit')

Usage (di setup):
const toast = useToastStore()
toast.success('Soal berhasil disimpan')
```

---

## Komponen Soal (Exam Components)

### QuestionDisplay.vue (wrapper)
```
Props:
  - question: Question
  - answer: ExamAnswer | null
  - isReadOnly: boolean (default: false)

Emits: answer-change

Logika: render komponen yang tepat berdasarkan question.type
  pg         -> QuestionPG.vue
  pgk        -> QuestionPGK.vue
  bs         -> QuestionTrueFalse.vue
  menjodohkan -> QuestionMatching.vue
  singkat    -> QuestionFillIn.vue
  matrix     -> QuestionMatrix.vue
  esai       -> QuestionEssay.vue

Usage:
<QuestionDisplay 
    :question="currentQuestion" 
    :answer="answers[currentQuestion.id]"
    @answer-change="handleAnswerChange"
/>
```

### QuestionPG.vue
```
Props:
  - options: QuestionOption[]
  - modelValue: string | null (selected key: 'A', 'B', etc.)
  - isReadOnly: boolean

Emits: update:modelValue
```

### QuestionPGK.vue
```
Props:
  - options: QuestionOption[]
  - modelValue: string[] | null (selected keys: ['A', 'C'])
  - isReadOnly: boolean

Emits: update:modelValue
```

### QuestionMatching.vue
```
Props:
  - options: MatchingOption[] (left + right pairs)
  - modelValue: Record<string, string> | null ({A: '3', B: '1'})
  - isReadOnly: boolean

Emits: update:modelValue
```

### QuestionMatrix.vue
```
Props:
  - columns: string[]
  - rows: MatrixRow[]
  - modelValue: Record<string, number> | null
  - isReadOnly: boolean

Emits: update:modelValue
```

### QuestionEssay.vue
```
Props:
  - modelValue: string | null
  - maxWords: number (optional)
  - isReadOnly: boolean

Emits: update:modelValue

Note: Gunakan textarea biasa (bukan rich text) untuk peserta.
Rich text hanya untuk guru saat membuat soal.
```

### ExamTimer.vue
```
Props:
  - endTime: Date (absolute waktu habis dari server)
  - warningThreshold: number (menit, default: 5)

Emits: time-up, warning

Logika:
  - Hitung countdown dari server end_time
  - Di bawah threshold: tampil merah + pulse
  - time-up: emit event, trigger auto-finish
```

### QuestionNavigator.vue
```
Props:
  - questions: NavigatorQuestion[]  [{id, number, isAnswered, isDoubtful}]
  - currentIndex: number

Emits: navigate

Usage:
<QuestionNavigator 
    :questions="navQuestions" 
    :current-index="currentIndex"
    @navigate="jumpToQuestion"
/>
```

---

## Komponen Layout

### AdminLayout.vue
```
Structure:
  - AppSidebar.vue (kiri, fixed)
  - AppHeader.vue (atas, sticky)
  - <router-view /> (konten utama)
```

### AppSidebar.vue
```
Props:
  - collapsed: boolean

Nav items dari router berdasarkan role user:
  Admin:    Dashboard, Users, Rombels, Subjects, Rooms, Settings
  Guru:     Dashboard, Bank Soal, Jadwal, Laporan
  Pengawas: Dashboard, Monitoring
  Peserta:  tidak pakai layout ini (pakai PesertaLayout.vue)
```

### PesertaLayout.vue
```
Simple layout tanpa sidebar:
  - AppHeader simple (logo + nama peserta + logout)
  - <router-view />
```

### ExamLayout.vue
```
Full-screen layout untuk halaman ujian:
  - ExamHeader.vue (timer, progress, nama ujian)
  - <router-view /> (konten soal)
  
No sidebar, no navigation, focused UX.
```

---

## Composables

### useExamTimer.ts
```typescript
export function useExamTimer(endTime: Date) {
    const remaining = ref(0)
    const isWarning = ref(false)
    const isExpired = ref(false)
    
    // Hitung dari server endTime, bukan dari durasi client
    // untuk mencegah manipulasi timer
    
    return { remaining, isWarning, isExpired, formattedTime }
}
```

### useWebSocket.ts
```typescript
export function useWebSocket(scheduleId: number, role: string) {
    const isConnected = ref(false)
    const students = ref<StudentStatus[]>([])
    
    function connect() { ... }
    function disconnect() { ... }
    function sendEvent(event: WsEvent) { ... }
    function onEvent(type: string, handler: (data: unknown) => void) { ... }
    
    return { isConnected, students, connect, disconnect, sendEvent, onEvent }
}
```

### useApi.ts
```typescript
// Wrapper Axios dengan interceptors
export function useApi() {
    const { token, logout } = useAuthStore()
    
    const api = axios.create({
        baseURL: import.meta.env.VITE_API_URL,
    })
    
    // Request interceptor: inject Bearer token
    // Response interceptor: handle 401 -> logout, 422 -> return errors
    
    return { api, get, post, put, patch, del }
}
```

### usePagination.ts
```typescript
export function usePagination(initialPage = 1, initialPerPage = 20) {
    const page = ref(initialPage)
    const perPage = ref(initialPerPage)
    const meta = ref<PaginationMeta | null>(null)
    
    function setMeta(newMeta: PaginationMeta) { ... }
    function changePage(newPage: number) { ... }
    
    return { page, perPage, meta, setMeta, changePage }
}
```

---

## Design Tokens (TypeScript)

```typescript
// src/types/index.ts

export interface Question {
    id: number
    type: QuestionType
    questionBody: string
    audioPath: string | null
    audioLimit: number
    options: QuestionOption[] | null
    sortOrder: number
    defaultMark: number
    stimulus: Stimulus | null
}

export type QuestionType = 'pg' | 'pgk' | 'bs' | 'menjodohkan' | 'singkat' | 'matrix' | 'esai'

export interface QuestionOption {
    key: string
    text: string
    image: string | null
}

export interface ExamAnswer {
    questionId: number
    answer: string | string[] | Record<string, string | number> | null
    isDoubtful: boolean
    score: number | null
}

export interface ExamSession {
    id: string           // hash ID
    scheduleId: number
    status: 'ongoing' | 'completed' | 'terminated'
    startTime: string
    examEndTime: string  // server-calculated absolute end time
    violationCount: number
    progress: {
        answered: number
        total: number
        doubtful: number
    }
}

export interface StudentStatus {
    userId: number
    name: string
    isOnline: boolean
    answeredCount: number
    totalQuestions: number
    violationCount: number
    status: 'ongoing' | 'completed' | 'terminated'
}
```
