# UI/UX Design - CBT Patra

**Tanggal:** 2026-03-05
**Framework:** Vue.js 3 + Vite + TypeScript

---

## Design System

### Color Palette

```css
/* colors.css */
:root {
    /* Primary - Blue (kepercayaan, profesional) */
    --color-primary-50:  #EFF6FF;
    --color-primary-100: #DBEAFE;
    --color-primary-500: #3B82F6;
    --color-primary-600: #2563EB;
    --color-primary-700: #1D4ED8;
    --color-primary-900: #1E3A8A;

    /* Neutral - Gray */
    --color-gray-50:  #F9FAFB;
    --color-gray-100: #F3F4F6;
    --color-gray-200: #E5E7EB;
    --color-gray-300: #D1D5DB;
    --color-gray-400: #9CA3AF;
    --color-gray-500: #6B7280;
    --color-gray-600: #4B5563;
    --color-gray-700: #374151;
    --color-gray-800: #1F2937;
    --color-gray-900: #111827;

    /* Status Colors */
    --color-success: #10B981;    /* green - completed, online */
    --color-warning: #F59E0B;    /* amber - active, ongoing */
    --color-danger:  #EF4444;    /* red - terminated, offline */
    --color-info:    #06B6D4;    /* cyan - info */

    /* Background */
    --bg-primary:   #FFFFFF;
    --bg-secondary: #F9FAFB;
    --bg-sidebar:   #1E3A5F;     /* dark navy sidebar */

    /* Text */
    --text-primary:   #111827;
    --text-secondary: #6B7280;
    --text-muted:     #9CA3AF;
    --text-white:     #FFFFFF;
}
```

### Typography

```css
/* Google Fonts: Inter */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');

:root {
    --font-sans: 'Inter', system-ui, -apple-system, sans-serif;
    --font-mono: 'JetBrains Mono', 'Fira Code', monospace;

    /* Font Sizes */
    --text-xs:   0.75rem;   /* 12px */
    --text-sm:   0.875rem;  /* 14px */
    --text-base: 1rem;      /* 16px */
    --text-lg:   1.125rem;  /* 18px */
    --text-xl:   1.25rem;   /* 20px */
    --text-2xl:  1.5rem;    /* 24px */
    --text-3xl:  1.875rem;  /* 30px */

    /* Soal ujian: lebih besar untuk readability */
    --text-question: 1.0625rem; /* 17px */
}
```

### Spacing & Layout

```css
:root {
    --spacing-1:  0.25rem;
    --spacing-2:  0.5rem;
    --spacing-3:  0.75rem;
    --spacing-4:  1rem;
    --spacing-6:  1.5rem;
    --spacing-8:  2rem;
    --spacing-12: 3rem;
    --spacing-16: 4rem;

    --border-radius-sm:  0.25rem;
    --border-radius:     0.5rem;
    --border-radius-lg:  0.75rem;
    --border-radius-xl:  1rem;
    --border-radius-full: 9999px;

    --shadow-sm:  0 1px 2px 0 rgb(0 0 0 / 0.05);
    --shadow:     0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1);
    --shadow-md:  0 4px 6px -1px rgb(0 0 0 / 0.1);
    --shadow-lg:  0 10px 15px -3px rgb(0 0 0 / 0.1);

    --sidebar-width: 260px;
    --header-height: 64px;
}
```

---

## Layout Struktur

### Admin/Guru Layout
```
+------------------+--------------------------------+
|    SIDEBAR       |         HEADER                 |
|    (260px)       |  [Logo] [Menu...] [User Avatar] |
| [Logo]           +--------------------------------+
| [Nav items]      |         CONTENT AREA           |
|   > Dashboard    |                                |
|   > Users        |  <router-view />               |
|   > Bank Soal    |                                |
|   > Jadwal       |                                |
|   > Laporan      |                                |
|   > Pengaturan   |                                |
|                  |                                |
+------------------+--------------------------------+
```

### Peserta - Exam Layout
```
+--------------------------------------------------+
|  [Nama Ujian]     [Timer: 45:32]   [Soal 5/40]  |
+--------------------------------------------------+
|                                                  |
|  PANEL SOAL (70%)       |  PANEL NAVIGASI (30%)  |
|                          |                        |
|  [nomor] [tipe]          |  [navigator grid]      |
|                          |  1  2  3  4  5         |
|  Teks soal...            |  6  7  8  9  10        |
|                          |  (warna: dijawab,      |
|  A. Opsi                 |   ragu, belum)         |
|  B. Opsi                 |                        |
|  C. Opsi                 |  [Selesai Ujian]       |
|  D. Opsi                 |                        |
|                          |                        |
|  [Prev] [Flag] [Next]    |                        |
+--------------------------------------------------+
```

### Pengawas Monitoring Layout
```
+--------------------------------------------------+
|  Monitoring: [Nama Jadwal]   Online: 28/35       |
+--------------------------------------------------+
|  [Filter Kelas] [Filter Status] [Search]         |
+--------------------------------------------------+
|  GRID PESERTA                                    |
|  +--------+ +--------+ +--------+               |
|  | [Foto] | | [Foto] | | [Foto] |               |
|  | Nama   | | Nama   | | Nama   |               |
|  | 35/40  | | 12/40  | | 40/40  |               |
|  | Hijau  | | Kuning | | Biru   |               |
|  +--------+ +--------+ +--------+               |
|                                                  |
|  Status: Hijau=Online, Kuning=Ragu, Merah=Off   |
+--------------------------------------------------+
```

---

## Status Badges

```
Ujian Akan Datang: [ Akan Datang ] - abu-abu
Ujian Berlangsung: [ Berlangsung ] - kuning (pulse animation)
Ujian Selesai:     [   Selesai   ] - hijau

Sesi Ongoing:      [ Mengerjakan ] - kuning
Sesi Completed:    [   Selesai   ] - hijau
Sesi Terminated:   [  Dihentikan ] - merah

Peserta Online:    [   Online    ] - hijau (pulsing dot)
Peserta Offline:   [   Offline   ] - merah
```

---

## Komponen Soal Per Tipe

### PG (Pilihan Ganda Biasa)
```
[ Nomor ] [ Tipe: PG ]

Teks pertanyaan yang bisa berisi HTML...

(o) A. Opsi pertama
( ) B. Opsi kedua  
( ) C. Opsi ketiga
( ) D. Opsi keempat

[Flag Ragu-ragu] [< Prev] [Next >]
```

### PGK (Pilihan Ganda Kompleks)
```
[ Nomor ] [ Tipe: PGK (pilih lebih dari satu) ]

Teks pertanyaan...

[x] A. Opsi pertama  (checkbox)
[ ] B. Opsi kedua
[x] C. Opsi ketiga
[ ] D. Opsi keempat

Terpilih: A, C
```

### Menjodohkan
```
[ Nomor ] [ Tipe: Menjodohkan ]

Pasangkan kolom kiri dengan kanan:

Kiri              Kanan
A. Indonesia   -> [dropdown: Jakarta ▾]
B. Jepang      -> [dropdown: Tokyo   ▾]
C. Perancis    -> [dropdown: ------  ▾]
```

### Matrix/Tabel
```
[ Nomor ] [ Tipe: Matrix ]

             | Fakta | Opini |
-------------|-------|-------|
Bumi bulat   | [x]   | [ ]   |
Nasi enak    | [ ]   | [x]   |
Air H2O      | [x]   | [ ]   |
```

### Esai
```
[ Nomor ] [ Tipe: Esai ]

Teks pertanyaan...

[   Textarea (rich text editor)              ]
[                                            ]
[   Tulis jawaban Anda di sini...           ]
[____________________________________________]

Kata: 0
```

---

## Pages

### Login Page
- Form sederhana: username + password
- Logo sekolah/instansi
- Background: gradient primary

### Peserta Dashboard
- Header: selamat datang, nama peserta
- Section: Ujian Aktif (card dengan timer)
- Section: Ujian Mendatang
- Section: Riwayat Ujian Selesai

### Halaman Ujian (ExamPage)
- Full-screen mode
- Header: nama ujian + timer + progress
- Left: tampilan soal (70%)
- Right: navigator + tombol selesai (30%)
- Auto-save setiap jawaban berubah
- Warning saat pindah tab (pelanggaran)

### Dashboard Guru - Bank Soal
- Tabel bank soal dengan search + filter
- CRUD soal in-place
- Preview soal

### Dashboard Admin - Manajemen User
- Tabel user dengan search + filter role
- Import Excel modal
- Edit user modal

### Dashboard Pengawas
- Grid peserta real-time
- WebSocket reconnect otomatis
- Alert ketika ada pelanggaran
- Tombol kunci klien per peserta

---

## Animasi & Micro-interactions

```css
/* Timer pulse (ujian berjalan) */
.badge-active {
    animation: pulse 2s infinite;
}

@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.6; }
}

/* Online indicator */
.online-dot {
    background: var(--color-success);
    animation: pulse-dot 2s infinite;
}

/* Violation warning shake */
.violation-alert {
    animation: shake 0.5s;
}

@keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-5px); }
    75% { transform: translateX(5px); }
}

/* Page transition */
.page-enter-active, .page-leave-active {
    transition: opacity 0.2s ease;
}
.page-enter-from, .page-leave-to {
    opacity: 0;
}
```

---

## Responsive Design

| Breakpoint | Layar | Behavior |
|------------|-------|----------|
| sm: 640px | Mobile | Sidebar collapse, bottom nav |
| md: 768px | Tablet | Sidebar mini (icon only) |
| lg: 1024px | Desktop | Full sidebar |
| xl: 1280px | Wide | Full sidebar + wider content |

**Catatan:** Halaman ujian (ExamPage) tidak perlu responsive yang ekstrem karena asumsi 
peserta ujian menggunakan laptop/komputer. Namun tetap functional di tablet.
