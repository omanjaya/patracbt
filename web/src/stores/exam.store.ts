import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useExamStore = defineStore('exam', () => {
  // Active session info
  const sessionId = ref<number | null>(null)
  const scheduleId = ref<number | null>(null)
  const questions = ref<any[]>([])
  const answersMap = ref<Map<number, any>>(new Map())
  const flaggedSet = ref<Set<number>>(new Set())
  const currentIndex = ref(0)
  const status = ref<'idle' | 'loading' | 'ongoing' | 'finished'>('idle')

  // Computed
  const currentQuestion = computed(() => questions.value[currentIndex.value] ?? null)
  const answeredCount = computed(() => {
    let count = 0
    answersMap.value.forEach((v) => { if (v !== null && v !== undefined) count++ })
    return count
  })
  const totalQuestions = computed(() => questions.value.length)
  const progressPercent = computed(() =>
    totalQuestions.value > 0 ? Math.round((answeredCount.value / totalQuestions.value) * 100) : 0
  )
  const unansweredQuestions = computed(() =>
    questions.value.filter((q) => !answersMap.value.has(q.id)).map((q) => ({ ...q, index: questions.value.indexOf(q) }))
  )
  const flaggedQuestions = computed(() =>
    questions.value.filter((q) => flaggedSet.value.has(q.id)).map((q) => ({ ...q, index: questions.value.indexOf(q) }))
  )

  // Actions
  function initSession(data: { id: number; schedule_id: number; questions: any[]; answers: any[] }) {
    sessionId.value = data.id
    scheduleId.value = data.schedule_id
    questions.value = data.questions
    answersMap.value = new Map()
    flaggedSet.value = new Set()
    for (const a of data.answers) {
      if (a.answer !== null) answersMap.value.set(a.question_id, a.answer)
      if (a.is_flagged) flaggedSet.value.add(a.question_id)
    }
    currentIndex.value = 0
    status.value = 'ongoing'
  }

  function setAnswer(questionId: number, answer: any) {
    answersMap.value = new Map(answersMap.value.set(questionId, answer))
  }

  function toggleFlag(questionId: number) {
    const newSet = new Set(flaggedSet.value)
    if (newSet.has(questionId)) newSet.delete(questionId)
    else newSet.add(questionId)
    flaggedSet.value = newSet
  }

  function goTo(index: number) {
    if (index >= 0 && index < questions.value.length) currentIndex.value = index
  }

  function reset() {
    sessionId.value = null
    scheduleId.value = null
    questions.value = []
    answersMap.value = new Map()
    flaggedSet.value = new Set()
    currentIndex.value = 0
    status.value = 'idle'
  }

  return {
    sessionId, scheduleId, questions, answersMap, flaggedSet, currentIndex, status,
    currentQuestion, answeredCount, totalQuestions, progressPercent, unansweredQuestions, flaggedQuestions,
    initSession, setAnswer, toggleFlag, goTo, reset,
  }
})
