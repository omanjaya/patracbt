import { createRouter, createWebHistory } from 'vue-router'
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth.store'

declare module 'vue-router' {
  interface RouteMeta {
    guest?: boolean
    requiresAuth?: boolean
    roles?: string[]
  }
}

/** Global flag indicating auth initialization is in progress */
export const isAuthChecking = ref(false)

let authInitPromise: Promise<void> | null = null

const router = createRouter({
    history: createWebHistory(),
    routes: [
        {
            path: '/login',
            name: 'Login',
            component: () => import('../pages/auth/LoginPage.vue'),
            meta: { guest: true },
        },
        {
            path: '/admin',
            component: () => import('../components/layout/AdminLayout.vue'),
            meta: { requiresAuth: true, roles: ['admin'] },
            children: [
                {
                    path: '',
                    name: 'AdminDashboard',
                    component: () => import('../pages/admin/dashboard/DashboardPage.vue'),
                },
                // Master Data
                {
                    path: 'rombels',
                    name: 'Rombels',
                    component: () => import('../pages/admin/rombels/RombelsPage.vue'),
                },
                {
                    path: 'rooms',
                    name: 'Rooms',
                    component: () => import('../pages/admin/rooms/RoomsPage.vue'),
                },
                {
                    path: 'tags',
                    name: 'Tags',
                    component: () => import('../pages/admin/tags/TagsPage.vue'),
                },
                {
                    path: 'subjects',
                    name: 'Subjects',
                    component: () => import('../pages/admin/subjects/SubjectsPage.vue'),
                },
                {
                    path: 'roles',
                    name: 'Roles',
                    component: () => import('../pages/admin/roles/RolesPage.vue'),
                },
                {
                    path: 'roles/user-permissions',
                    name: 'UserPermissions',
                    component: () => import('../pages/admin/roles/UserPermissionsPage.vue'),
                },
                {
                    path: 'roles/master-permissions',
                    name: 'MasterPermissions',
                    component: () => import('../pages/admin/roles/MasterPermissionsPage.vue'),
                },
                // User Management
                {
                    path: 'users',
                    name: 'Users',
                    component: () => import('../pages/admin/users/UsersPage.vue'),
                },
                {
                    path: 'users/trash',
                    name: 'UsersTrash',
                    component: () => import('../pages/admin/users/UsersTrashPage.vue'),
                },
                {
                    path: 'rombel-management',
                    name: 'RombelManagement',
                    component: () => import('../pages/admin/rombel-management/RombelManagementPage.vue'),
                },
                {
                    path: 'room-management',
                    name: 'RoomManagement',
                    component: () => import('../pages/admin/room-management/RoomManagementPage.vue'),
                },
                {
                    path: 'user-tags',
                    name: 'UserTags',
                    component: () => import('../pages/admin/user-tags/UserTagsPage.vue'),
                },
                {
                    path: 'print-cards',
                    name: 'PrintCards',
                    component: () => import('../pages/admin/print-cards/PrintCardsPage.vue'),
                },
                {
                    path: 'print-cards/setup',
                    name: 'PrintCardsSetup',
                    component: () => import('../pages/admin/print-cards/PrintSetupPage.vue'),
                },
                // Exam Management
                {
                    path: 'question-banks',
                    name: 'QuestionBanks',
                    component: () => import('../pages/admin/question-banks/QuestionBanksPage.vue'),
                },
                {
                    path: 'question-banks/:id',
                    name: 'QuestionBankDetail',
                    component: () => import('../pages/admin/question-banks/QuestionBankDetailPage.vue'),
                },
                {
                    path: 'question-banks/:id/import',
                    name: 'QuestionsImport',
                    component: () => import('../pages/admin/question-banks/QuestionsImportPage.vue'),
                },
                {
                    path: 'question-banks/:id/print',
                    name: 'QuestionsPrint',
                    component: () => import('../pages/admin/question-banks/QuestionsPrintPage.vue'),
                },
                {
                    path: 'exam-schedules',
                    name: 'ExamSchedules',
                    component: () => import('../pages/admin/exam-schedules/ExamSchedulesPage.vue'),
                },
                {
                    path: 'exam-schedules/create',
                    name: 'ExamScheduleCreate',
                    component: () => import('../pages/admin/exam-schedules/ExamScheduleFormPage.vue'),
                },
                {
                    path: 'exam-schedules/:id/edit',
                    name: 'ExamScheduleEdit',
                    component: () => import('../pages/admin/exam-schedules/ExamScheduleFormPage.vue'),
                },
                {
                    path: 'exam-schedules/:id/preview',
                    name: 'ExamSchedulePreview',
                    component: () => import('../pages/admin/exam-schedules/ExamSchedulePreviewPage.vue'),
                },
                {
                    path: 'exam-schedules/trash',
                    name: 'ExamSchedulesTrash',
                    component: () => import('../pages/admin/exam-schedules/ExamSchedulesTrashPage.vue'),
                },
                {
                    path: 'supervision',
                    name: 'Supervision',
                    component: () => import('../pages/admin/supervision/SupervisionPage.vue'),
                },
                {
                    path: 'supervision/claim',
                    name: 'SupervisionClaim',
                    component: () => import('../pages/admin/supervision/SupervisionClaimPage.vue'),
                },
                {
                    path: 'supervision/global',
                    name: 'SupervisionGlobal',
                    component: () => import('../pages/admin/supervision/SupervisionGlobalPage.vue'),
                },
                {
                    path: 'reports',
                    name: 'Reports',
                    component: () => import('../pages/admin/reports/ReportsPage.vue'),
                },
                {
                    path: 'reports/ledger',
                    name: 'ReportsLedger',
                    component: () => import('../pages/admin/reports/ReportsLedgerPage.vue'),
                },
                {
                    path: 'live-score',
                    name: 'LiveScore',
                    component: () => import('../pages/admin/live-score/LiveScorePage.vue'),
                },
                // Settings
                {
                    path: 'settings',
                    name: 'Settings',
                    component: () => import('../pages/admin/settings/SettingsPage.vue'),
                },
                {
                    path: 'settings/database',
                    name: 'DatabaseManagement',
                    component: () => import('../pages/admin/settings/DatabaseManagementPage.vue'),
                },
                {
                    path: 'profile',
                    name: 'AdminProfile',
                    component: () => import('../pages/profile/ProfilePage.vue'),
                },
            ],
        },
        {
            path: '/guru',
            component: () => import('../components/layout/AdminLayout.vue'),
            meta: { requiresAuth: true, roles: ['guru', 'admin'] },
            children: [
                {
                    path: '',
                    name: 'GuruDashboard',
                    component: () => import('../pages/guru/dashboard/DashboardPage.vue'),
                },
                {
                    path: 'question-banks',
                    name: 'GuruQuestionBanks',
                    component: () => import('../pages/admin/question-banks/QuestionBanksPage.vue'),
                },
                {
                    path: 'question-banks/:id',
                    name: 'GuruQuestionBankDetail',
                    component: () => import('../pages/admin/question-banks/QuestionBankDetailPage.vue'),
                },
                {
                    path: 'question-banks/:id/import',
                    name: 'GuruQuestionsImport',
                    component: () => import('../pages/admin/question-banks/QuestionsImportPage.vue'),
                },
                {
                    path: 'question-banks/:id/print',
                    name: 'GuruQuestionsPrint',
                    component: () => import('../pages/admin/question-banks/QuestionsPrintPage.vue'),
                },
                {
                    path: 'exam-schedules',
                    name: 'GuruExamSchedules',
                    component: () => import('../pages/admin/exam-schedules/ExamSchedulesPage.vue'),
                },
                {
                    path: 'exam-schedules/create',
                    name: 'GuruExamScheduleCreate',
                    component: () => import('../pages/admin/exam-schedules/ExamScheduleFormPage.vue'),
                },
                {
                    path: 'exam-schedules/:id/edit',
                    name: 'GuruExamScheduleEdit',
                    component: () => import('../pages/admin/exam-schedules/ExamScheduleFormPage.vue'),
                },
                {
                    path: 'exam-schedules/:id/preview',
                    name: 'GuruExamSchedulePreview',
                    component: () => import('../pages/admin/exam-schedules/ExamSchedulePreviewPage.vue'),
                },
                {
                    path: 'exam-schedules/trash',
                    name: 'GuruExamSchedulesTrash',
                    component: () => import('../pages/admin/exam-schedules/ExamSchedulesTrashPage.vue'),
                },
                {
                    path: 'reports',
                    name: 'GuruReports',
                    component: () => import('../pages/guru/reports/GuruReportsPage.vue'),
                },
                {
                    path: 'reports/ledger',
                    name: 'GuruReportsLedger',
                    component: () => import('../pages/admin/reports/ReportsLedgerPage.vue'),
                },
                {
                    path: 'exam-history',
                    name: 'GuruExamHistory',
                    component: () => import('../pages/guru/exam-history/ExamHistoryPage.vue'),
                },
                {
                    path: 'supervision',
                    name: 'GuruSupervision',
                    component: () => import('../pages/admin/supervision/SupervisionPage.vue'),
                },
                {
                    path: 'live-score',
                    name: 'GuruLiveScore',
                    component: () => import('../pages/admin/live-score/LiveScorePage.vue'),
                },
                {
                    path: 'grading/:scheduleId',
                    name: 'GradingList',
                    component: () => import('../pages/guru/grading/GradingPage.vue'),
                },
                {
                    path: 'grading/:scheduleId/:userId/:sessionId',
                    name: 'GradingDetail',
                    component: () => import('../pages/guru/grading/GradingDetailPage.vue'),
                },
                {
                    path: 'profile',
                    name: 'GuruProfile',
                    component: () => import('../pages/profile/ProfilePage.vue'),
                },
            ],
        },
        {
            path: '/pengawas',
            component: () => import('../components/layout/AdminLayout.vue'),
            meta: { requiresAuth: true, roles: ['pengawas', 'admin'] },
            children: [
                {
                    path: '',
                    name: 'PengawasDashboard',
                    component: () => import('../pages/pengawas/dashboard/DashboardPage.vue'),
                },
                {
                    path: 'supervision',
                    name: 'PengawasSupervision',
                    component: () => import('../pages/pengawas/supervision/SupervisionHubPage.vue'),
                },
                {
                    path: 'supervision/:scheduleId',
                    name: 'PengawasSupervisionDetail',
                    component: () => import('../pages/admin/supervision/SupervisionPage.vue'),
                },
                {
                    path: 'violations',
                    name: 'PengawasViolations',
                    component: () => import('../pages/pengawas/violations/ViolationLogPage.vue'),
                },
                {
                    path: 'live-score',
                    name: 'PengawasLiveScore',
                    component: () => import('../pages/admin/live-score/LiveScorePage.vue'),
                },
                {
                    path: 'reports',
                    name: 'PengawasReports',
                    component: () => import('../pages/admin/reports/ReportsPage.vue'),
                },
                {
                    path: 'profile',
                    name: 'PengawasProfile',
                    component: () => import('../pages/profile/ProfilePage.vue'),
                },
            ],
        },
        {
            path: '/peserta',
            component: () => import('../components/layout/PesertaLayout.vue'),
            meta: { requiresAuth: true, roles: ['peserta'] },
            children: [
                {
                    path: '',
                    name: 'PesertaDashboard',
                    component: () => import('../pages/peserta/dashboard/DashboardPage.vue'),
                },
                {
                    path: 'confirm/:id',
                    name: 'ExamConfirm',
                    component: () => import('../pages/peserta/exam/ConfirmPage.vue'),
                },
                {
                    path: 'profile',
                    name: 'PesertaProfile',
                    component: () => import('../pages/profile/ProfilePage.vue'),
                },
            ],
        },
        // Standalone exam routes (no layout)
        {
            path: '/peserta/exam/:id',
            name: 'ExamPage',
            component: () => import('../pages/peserta/exam/ExamPage.vue'),
            meta: { requiresAuth: true, roles: ['peserta'] },
        },
        {
            path: '/peserta/exam/:id/locked',
            name: 'ExamLocked',
            component: () => import('../pages/peserta/exam/ExamLockedPage.vue'),
            meta: { requiresAuth: true, roles: ['peserta'] },
        },
        {
            path: '/peserta/exam/:id/transition',
            name: 'ExamTransition',
            component: () => import('../pages/peserta/exam/TransitionPage.vue'),
            meta: { requiresAuth: true, roles: ['peserta'] },
        },
        {
            path: '/peserta/result/:id',
            name: 'ResultPage',
            component: () => import('../pages/peserta/exam/ResultPage.vue'),
            meta: { requiresAuth: true, roles: ['peserta'] },
        },
        {
            path: '/peserta/report/:sessionId',
            name: 'peserta-report',
            component: () => import('../pages/peserta/report/PersonalReportPage.vue'),
            meta: { requiresAuth: true, roles: ['peserta'] },
        },
        {
            path: '/403',
            name: 'Error403',
            component: () => import('../pages/error/Error403.vue'),
        },
        {
            path: '/404',
            name: 'Error404',
            component: () => import('../pages/error/Error404.vue'),
        },
        {
            path: '/500',
            name: 'Error500',
            component: () => import('../pages/error/Error500.vue'),
        },
        {
            path: '/offline',
            name: 'Offline',
            component: () => import('../pages/errors/OfflinePage.vue'),
        },
        {
            path: '/pwa-required',
            name: 'PwaRequired',
            component: () => import('../pages/errors/PwaRequiredPage.vue'),
        },
        {
            path: '/limited-mode',
            name: 'LimitedMode',
            component: () => import('../pages/errors/LimitedModePage.vue'),
        },
        {
            path: '/',
            redirect: '/login',
        },
        {
            path: '/:pathMatch(.*)*',
            redirect: '/404',
        },
    ],
})

router.beforeEach(async (to, _from, next) => {
    const authStore = useAuthStore()

    // Prevent race conditions: reuse existing auth init promise
    if (!authStore.user && localStorage.getItem('access_token')) {
        if (!authInitPromise) {
            isAuthChecking.value = true
            authInitPromise = authStore.init()
                .catch(() => {
                    // Auth init failed (network error, invalid token, etc.)
                    localStorage.removeItem('access_token')
                    localStorage.removeItem('refresh_token')
                })
                .finally(() => {
                    isAuthChecking.value = false
                    authInitPromise = null
                })
        }
        await authInitPromise
    }

    if (to.meta.guest && authStore.isAuthenticated) {
        authStore.user && next(getRoleHome(authStore.user.role))
        return
    }

    if (to.meta.requiresAuth && !authStore.isAuthenticated) {
        next('/login')
        return
    }

    if (to.meta.roles && authStore.user) {
        if (!to.meta.roles.includes(authStore.user.role)) {
            next('/403')
            return
        }
    }

    next()
})

function getRoleHome(role: string): string {
    const map: Record<string, string> = {
        admin: '/admin',
        guru: '/guru',
        pengawas: '/pengawas',
        peserta: '/peserta',
    }
    return map[role] ?? '/login'
}

export default router
