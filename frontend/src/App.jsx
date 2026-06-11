import { useEffect, useMemo, useState } from 'react'
import {
  ArrowDown,
  ArrowLeft,
  BarChart3,
  Bell,
  Bot,
  CalendarDays,
  CameraOff,
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Clock3,
  Contact,
  Copy,
  Download,
  Edit3,
  ExternalLink,
  FileText,
  Filter,
  Folder,
  Grid2X2,
  Highlighter,
  Link,
  ListChecks,
  Loader2,
  Lock,
  MessageSquareText,
  Mic,
  MicOff,
  MoreHorizontal,
  PanelRight,
  Play,
  Radio,
  RefreshCw,
  Search,
  Send,
  Settings,
  Share2,
  ShieldCheck,
  Sparkles,
  Trash2,
  TrendingUp,
  Users,
  Video,
  Volume2,
  Zap,
} from 'lucide-react'
import { LiveKitRoom, VideoConference, useParticipants } from '@livekit/components-react'
import '@livekit/components-styles'
import './App.css'

const navItems = [
  { id: 'meeting', label: 'AlemLive', icon: Grid2X2 },
  { id: 'reports', label: 'Отчёты', icon: FileText },
]

const reportTabs = [
  { id: 'notes', label: 'Заметки', icon: FileText },
  { id: 'transcript', label: 'Транскрипт', icon: MessageSquareText },
  { id: 'deepDive', label: 'Глубокое погружение', icon: BarChart3 },
  { id: 'highlights', label: 'Основные моменты', icon: Highlighter },
  { id: 'chapters', label: 'Главы', icon: ListChecks },
]

const reportRows = [
  {
    id: 'read-intro',
    title: 'Ввод в Alem AI - Пример отчёта',
    source: 'Google Meet',
    date: 'пт, 2 янв. 2026 г.',
    time: '02:00 - 03:45',
    participants: 4,
    score: 89,
    folder: 'Образцы отчётов',
    owner: 'Мади',
    ownerInitial: 'М',
    thumbnailTone: 'teal',
    week: 'НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025',
  },
  {
    id: 'meeting-usage',
    title: 'Использование отчёта собрания - Пример отчёта',
    source: 'Google Meet',
    date: 'пт, 2 янв. 2026 г.',
    time: '01:00 - 01:04',
    participants: 4,
    score: 89,
    folder: 'Образцы отчётов',
    owner: 'Айдана',
    ownerInitial: 'А',
    thumbnailTone: 'blue',
    week: 'НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025',
  },
  {
    id: 'copilot-search',
    title: 'Используйте Copilot для поиска - Пример отчёта',
    source: 'Google Meet',
    date: 'пт, 2 янв. 2026 г.',
    time: '00:00 - 00:07',
    participants: 4,
    score: 88,
    folder: 'Образцы отчётов',
    owner: 'Елиас',
    ownerInitial: 'Е',
    thumbnailTone: 'violet',
    week: 'НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025',
  },
  {
    id: 'mobile-guide',
    title: 'Руководство по использованию настольного и мобильного приложения',
    source: 'Google Meet',
    date: 'чт, 1 янв. 2026 г.',
    time: '23:00 - 23:04',
    participants: 5,
    score: 92,
    folder: 'Образцы отчётов',
    owner: 'Келси',
    ownerInitial: 'К',
    thumbnailTone: 'green',
    week: 'НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025',
  },
  {
    id: 'real-cases',
    title: 'Исследуйте реальные случаи использования - Пример отчёта',
    source: 'Google Meet',
    date: 'чт, 1 янв. 2026 г.',
    time: '22:00 - 22:08',
    participants: 4,
    score: 87,
    folder: 'Образцы отчётов',
    owner: 'Сара',
    ownerInitial: 'С',
    thumbnailTone: 'rose',
    week: 'НЕДЕЛЯ С 29 ДЕК.-4 ЯНВ., 2025',
  },
]

const actionItems = [
  { task: 'Подготовить список вопросов для демо клиента', owner: 'Мади Орысбек', due: 'Сегодня, 18:00' },
  { task: 'Проверить backend endpoint для LiveKit token', owner: 'Айдана Сейт', due: 'Завтра, 11:00' },
  { task: 'Обновить UI отчёта после тестовой встречи', owner: 'Team AI', due: 'После созвона' },
]

const transcriptLines = [
  {
    time: '00:42',
    speaker: 'Мади',
    text: 'Нам нужно, чтобы участник мог войти в комнату только по названию, без ручного token.',
  },
  {
    time: '04:18',
    speaker: 'Айдана',
    text: 'После встречи отчёт должен быстро показывать summary, задачи и полный контекст разговора.',
  },
  {
    time: '12:05',
    speaker: 'Team AI',
    text: 'Я выделю главы, вопросы и места, где обсуждение затянулось или было особенно активным.',
  },
]

const speakerStats = [
  { name: 'Мади', talk: 48, sentiment: 'Позитивный', pace: '142 слов/мин' },
  { name: 'Айдана', talk: 34, sentiment: 'Нейтральный', pace: '128 слов/мин' },
  { name: 'Team AI', talk: 18, sentiment: 'Фокус', pace: '96 слов/мин' },
]

const highlights = [
  { time: '03:20', title: 'Решение по входу в комнату', note: 'Название комнаты становится главным способом подключения.' },
  { time: '17:45', title: 'Риск по backend', note: 'Если backend не запущен, агент должен явно показать ошибку подключения.' },
  { time: '28:10', title: 'Следующий шаг', note: 'Добавить автоматический отчёт после завершения митинга.' },
]

const chapters = [
  { time: '00:00', title: 'Старт и цель встречи', duration: '4 мин' },
  { time: '04:01', title: 'LiveKit вход и комнаты', duration: '9 мин' },
  { time: '13:10', title: 'Структура AI отчёта', duration: '12 мин' },
  { time: '25:30', title: 'Action items и финальные решения', duration: '7 мин' },
]

const aiQuestions = [
  'Как отключить автоматическую отправку заметок внешним участникам?',
  'Как проверить и отредактировать заметки перед отправкой?',
  'Какие права нужны для Search Copilot?',
  'Какие задачи появились после встречи?',
  'Переведите резюме встречи на русский.',
]

const reportCalendarToday = new Date(2026, 5, 12)

const quickDateOptions = [
  { id: 'all', label: 'В любое время' },
  { id: 'today', label: 'Сегодня', days: 1 },
  { id: 'last7', label: 'Последние 7 дней', days: 7 },
  { id: 'last30', label: 'Последние 30 дней', days: 30 },
  { id: 'last90', label: 'Последние 90 дней', days: 90 },
  { id: 'last6months', label: 'Последние 6 месяцев', months: 6 },
  { id: 'last12months', label: 'Последние 12 месяцев', months: 12 },
]

const typeFilterOptions = [
  { id: 'meetings', label: 'Отчеты о встречах' },
  { id: 'readout', label: 'Темы Readout' },
  { id: 'daily', label: 'Ежедневные обзоры' },
]

const calendarMonthNames = [
  'январь',
  'февраль',
  'март',
  'апрель',
  'май',
  'июнь',
  'июль',
  'август',
  'сентябрь',
  'октябрь',
  'ноябрь',
  'декабрь',
]

const calendarShortMonthNames = ['янв.', 'фев.', 'мар.', 'апр.', 'мая', 'июн.', 'июл.', 'авг.', 'сен.', 'окт.', 'ноя.', 'дек.']
const calendarWeekdays = ['MO', 'TU', 'WE', 'TH', 'FR', 'SA', 'SU']

function normalizeDate(date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function shiftDate(date, amount) {
  const nextDate = new Date(date)
  nextDate.setDate(nextDate.getDate() + amount)
  return nextDate
}

function shiftMonth(date, amount) {
  return new Date(date.getFullYear(), date.getMonth() + amount, 1)
}

function getDateKey(date) {
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`
}

function isSameDate(firstDate, secondDate) {
  return firstDate && secondDate && firstDate.getTime() === secondDate.getTime()
}

function isBetweenDates(date, startDate, endDate) {
  return startDate && endDate && date > startDate && date < endDate
}

function getQuickDateRange(option) {
  const endDate = normalizeDate(reportCalendarToday)

  if (option.id === 'all') {
    return { from: null, to: null }
  }

  if (option.months) {
    return {
      from: new Date(endDate.getFullYear(), endDate.getMonth() - option.months, endDate.getDate()),
      to: endDate,
    }
  }

  return {
    from: shiftDate(endDate, -(option.days - 1)),
    to: endDate,
  }
}

function formatCalendarDate(date) {
  return `${date.getDate()} ${calendarShortMonthNames[date.getMonth()]}`
}

function formatDateRange(range) {
  if (!range.from) {
    return quickDateOptions[0].label
  }

  if (!range.to || isSameDate(range.from, range.to)) {
    return formatCalendarDate(range.from)
  }

  return `${formatCalendarDate(range.from)} - ${formatCalendarDate(range.to)}`
}

function getCalendarDays(monthDate) {
  const monthStart = new Date(monthDate.getFullYear(), monthDate.getMonth(), 1)
  const startOffset = (monthStart.getDay() + 6) % 7
  const gridStart = shiftDate(monthStart, -startOffset)

  return Array.from({ length: 42 }, (_, index) => {
    const date = normalizeDate(shiftDate(gridStart, index))

    return {
      date,
      key: getDateKey(date),
      isCurrentMonth: date.getMonth() === monthDate.getMonth(),
      isDisabled: date > normalizeDate(reportCalendarToday),
    }
  })
}

function formatAPIDate(date) {
  if (!date) {
    return ''
  }

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

async function apiRequest(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
  })

  const contentType = response.headers.get('content-type') || ''
  const payload = contentType.includes('application/json') ? await response.json().catch(() => ({})) : await response.text()

  if (!response.ok) {
    throw new Error(payload?.error || payload?.message || 'Backend request failed')
  }

  return payload
}

function buildReportsQuery({ search, mode, timeFilterMode, timeFilterRange }) {
  const params = new URLSearchParams()

  if (search.trim()) {
    params.set('q', search.trim())
  }

  if (mode === 'incomplete') {
    params.set('mode', 'incomplete')
  }

  if (timeFilterMode === 'custom' && timeFilterRange.from) {
    params.set('from', formatAPIDate(timeFilterRange.from))
    params.set('to', formatAPIDate(timeFilterRange.to || timeFilterRange.from))
  } else if (timeFilterMode && timeFilterMode !== 'all') {
    params.set('datePreset', timeFilterMode)
  }

  return params.toString()
}

function getInitialReportId() {
  if (typeof window === 'undefined') {
    return ''
  }

  const [, reportId] = window.location.hash.match(/^#report\/(.+)$/) || []
  return reportId || ''
}

function getParticipantRole(participant) {
  try {
    const metadata = JSON.parse(participant.metadata || '{}')
    if (metadata.role === 'host') {
      return 'Host'
    }
  } catch {
    // ignore malformed metadata
  }

  return 'Участник'
}

function ParticipantsList({ participants }) {
  return (
    <section className="panel participants-panel">
      <div className="panel-heading">
        <span className="panel-icon">
          <Contact size={21} />
        </span>
        <div>
          <h2>Участники</h2>
          <p>Команда встречи</p>
        </div>
      </div>

      <div className="member-list">
        {participants.length === 0 ? (
          <p className="empty-members">Пока никто не присоединился</p>
        ) : (
          participants.map((participant) => {
            const name = participant.name || participant.identity
            return (
              <div className="member" key={participant.identity}>
                <span className="member-avatar">{name.slice(0, 1).toUpperCase()}</span>
                <div>
                  <strong>{name}</strong>
                  <small>{getParticipantRole(participant)}</small>
                </div>
              </div>
            )
          })
        )}
      </div>
    </section>
  )
}

function ParticipantsPanel() {
  const participants = useParticipants()
  return <ParticipantsList participants={participants} />
}

function App() {
  const initialReportId = getInitialReportId()
  const [activeView, setActiveView] = useState(initialReportId ? 'reportDetail' : 'reports')
  const [selectedReportId, setSelectedReportId] = useState(initialReportId || reportRows[0].id)
  const [activeReportTab, setActiveReportTab] = useState('notes')
  const [form, setForm] = useState({
    roomName: import.meta.env.VITE_LIVEKIT_ROOM ?? 'alem-meeting',
    userName: import.meta.env.VITE_LIVEKIT_NAME ?? 'Мади Орысбек',
  })
  const [meeting, setMeeting] = useState(null)
  const [entryMode, setEntryMode] = useState('create')
  const [devices, setDevices] = useState(() => {
    if (typeof window === 'undefined') {
      return { mic: true, camera: true }
    }

    try {
      const saved = JSON.parse(window.localStorage.getItem('alemlive-devices'))
      return {
        mic: saved?.mic ?? true,
        camera: saved?.camera ?? true,
      }
    } catch {
      return { mic: true, camera: true }
    }
  })
  const [isStarting, setIsStarting] = useState(false)
  const [joinError, setJoinError] = useState('')
  const [isTimeFilterOpen, setIsTimeFilterOpen] = useState(false)
  const [timeFilterMode, setTimeFilterMode] = useState('all')
  const [timeFilterRange, setTimeFilterRange] = useState({ from: null, to: null })
  const [draftTimeFilterRange, setDraftTimeFilterRange] = useState({ from: null, to: null })
  const [calendarMonth, setCalendarMonth] = useState(new Date(reportCalendarToday.getFullYear(), reportCalendarToday.getMonth(), 1))
  const [isTypeFilterOpen, setIsTypeFilterOpen] = useState(false)
  const [selectedTypeFilterIds, setSelectedTypeFilterIds] = useState(typeFilterOptions.map((option) => option.id))
  const [openReportActionsId, setOpenReportActionsId] = useState('')
  const [profile, setProfile] = useState(null)
  const [notifications, setNotifications] = useState({ unread: 0, items: [] })
  const [locales, setLocales] = useState({ current: 'ru', items: [] })
  const [reports, setReports] = useState(reportRows)
  const [reportFilters, setReportFilters] = useState(null)
  const [reportDetails, setReportDetails] = useState({})
  const [reportActions, setReportActions] = useState({})
  const [reportsError, setReportsError] = useState('')
  const [reportsLoading, setReportsLoading] = useState(false)
  const [activeReportMode, setActiveReportMode] = useState('reports')
  const [reportSearchText, setReportSearchText] = useState('')
  const [workspaceNotice, setWorkspaceNotice] = useState('')
  const [reportActionMessage, setReportActionMessage] = useState('')
  const [copilotInput, setCopilotInput] = useState('')
  const [copilotMessages, setCopilotMessages] = useState([])
  const [isCopilotSending, setIsCopilotSending] = useState(false)
  const [isDetailActionsOpen, setIsDetailActionsOpen] = useState(false)

  const canStart = form.userName.trim() && form.roomName.trim()
  const isConnected = Boolean(meeting)
  const selectedReportDetail = reportDetails[selectedReportId]
  const selectedReport = selectedReportDetail?.report || reports.find((report) => report.id === selectedReportId) || reports[0] || reportRows[0]
  const dateFilterOptions = quickDateOptions.map((option) => ({
    ...option,
    label: reportFilters?.quickDateOptions?.find((backendOption) => backendOption.id === option.id)?.label || option.label,
  }))
  const activeQuickDateOption = dateFilterOptions.find((option) => option.id === timeFilterMode)
  const timeFilterLabel = timeFilterMode === 'custom' ? formatDateRange(timeFilterRange) : activeQuickDateOption?.label || quickDateOptions[0].label
  const calendarDays = getCalendarDays(calendarMonth)
  const areAllTypeFiltersSelected = selectedTypeFilterIds.length === typeFilterOptions.length
  const visibleReports = reports.length ? reports : reportRows

  const meetingMeta = useMemo(() => {
    const room = meeting?.roomName || form.roomName || 'alem-meeting'
    const name = meeting?.userName || form.userName || profile?.name || 'Guest'

    return {
      room,
      name,
      initial: profile?.initial || name.trim().slice(0, 1).toUpperCase() || 'M',
    }
  }, [form.roomName, form.userName, meeting, profile])

  useEffect(() => {
    let isMounted = true

    async function loadWorkspace() {
      const [profilePayload, notificationsPayload, localesPayload] = await Promise.allSettled([
        apiRequest('/api/profile'),
        apiRequest('/api/notifications'),
        apiRequest('/api/locales'),
      ])

      if (!isMounted) {
        return
      }

      if (profilePayload.status === 'fulfilled') {
        setProfile(profilePayload.value)
        setForm((current) => ({
          ...current,
          userName: current.userName || profilePayload.value.name || current.userName,
        }))
      }

      if (notificationsPayload.status === 'fulfilled') {
        setNotifications(notificationsPayload.value)
      }

      if (localesPayload.status === 'fulfilled') {
        setLocales(localesPayload.value)
      }
    }

    loadWorkspace().catch(() => {
      if (isMounted) {
        setWorkspaceNotice('Backend workspace недоступен, показаны локальные данные')
      }
    })

    return () => {
      isMounted = false
    }
  }, [])

  useEffect(() => {
    let isMounted = true
    const query = buildReportsQuery({ search: reportSearchText, mode: activeReportMode, timeFilterMode, timeFilterRange })

    async function loadReports() {
      setReportsLoading(true)
      setReportsError('')

      try {
        const payload = await apiRequest(`/api/reports${query ? `?${query}` : ''}`)
        if (!isMounted) {
          return
        }

        const nextReports = payload.reports || payload.items || []
        setReports(nextReports.length ? nextReports : [])
        setReportFilters(payload.filters || null)
        if (nextReports.length && !nextReports.some((report) => report.id === selectedReportId)) {
          setSelectedReportId(nextReports[0].id)
        }
      } catch (error) {
        if (isMounted) {
          setReports(reportRows)
          setReportsError(error.message || 'Не удалось загрузить отчёты из backend')
        }
      } finally {
        if (isMounted) {
          setReportsLoading(false)
        }
      }
    }

    loadReports()

    return () => {
      isMounted = false
    }
  }, [activeReportMode, reportSearchText, selectedReportId, timeFilterMode, timeFilterRange])

  useEffect(() => {
    if (!selectedReportId) {
      return undefined
    }

    let isMounted = true

    async function loadReportDetail() {
      try {
        const [detailPayload, actionsPayload] = await Promise.all([
          apiRequest(`/api/reports/${selectedReportId}`),
          apiRequest(`/api/reports/${selectedReportId}/actions`),
        ])

        if (!isMounted) {
          return
        }

        setReportDetails((current) => ({ ...current, [selectedReportId]: detailPayload }))
        setReportActions((current) => ({ ...current, [selectedReportId]: actionsPayload }))
      } catch {
        // Keep fallback report detail content when backend detail is unavailable.
      }
    }

    loadReportDetail()

    return () => {
      isMounted = false
    }
  }, [selectedReportId])

  function updateField(event) {
    const { name, value } = event.target
    setForm((current) => ({ ...current, [name]: value }))
    setJoinError('')
  }

  function selectEntryMode(mode) {
    setEntryMode(mode)
    setJoinError('')
  }

  function toggleDevice(name) {
    setDevices((current) => {
      const next = { ...current, [name]: !current[name] }
      window.localStorage.setItem('alemlive-devices', JSON.stringify(next))

      apiRequest('/api/devices', {
        method: 'POST',
        body: JSON.stringify({
          roomName: form.roomName || meetingMeta.room,
          userName: form.userName || meetingMeta.name,
          device: name,
          enabled: next[name],
        }),
      }).catch(() => {})

      return next
    })
  }

  function recordMeetingEvent(event) {
    return apiRequest('/api/meetings/events', {
      method: 'POST',
      body: JSON.stringify({
        roomName: meetingMeta.room,
        userName: meetingMeta.name,
        event,
      }),
    }).catch(() => null)
  }

  async function requestToken(roomName, userName, isHost) {
    const response = await fetch('/api/livekit/token', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ roomName, userName, isHost }),
    })

    const payload = await response.json().catch(() => ({}))

    if (!response.ok) {
      throw new Error(payload.error || 'Не удалось получить token для комнаты')
    }

    if (!payload.serverUrl || !payload.token) {
      throw new Error('Backend не вернул LiveKit URL или token')
    }

    return payload
  }

  async function startMeeting(mode = entryMode) {
    if (isStarting) {
      return
    }

    const nextRoomName = form.roomName.trim()
    const nextUserName = form.userName.trim()

    if (!nextUserName || !nextRoomName) {
      setJoinError('Введите имя и название комнаты')
      return
    }

    setIsStarting(true)
    setJoinError('')

    try {
      const isHost = mode === 'create'
      const payload = await requestToken(nextRoomName, nextUserName, isHost)

      setForm((current) => ({
        ...current,
        roomName: payload.roomName || nextRoomName,
        userName: payload.userName || nextUserName,
      }))
      setMeeting({
        serverUrl: payload.serverUrl,
        token: payload.token,
        roomName: payload.roomName || nextRoomName,
        userName: payload.userName || nextUserName,
        entryMode: mode,
        isHost,
        audio: devices.mic,
        video: devices.camera,
      })
      recordMeetingEvent(mode === 'create' ? 'created' : 'joined')
    } catch (error) {
      setJoinError(error.message)
    } finally {
      setIsStarting(false)
    }
  }

  function joinMeeting(event) {
    event.preventDefault()
    startMeeting(entryMode)
  }

  async function leaveMeeting() {
    await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/leave`, { method: 'POST' }).catch(() => null)
    await recordMeetingEvent('left')
    setMeeting(null)
  }

  async function copyRoom() {
    if (!navigator.clipboard) {
      return
    }

    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/link`).catch(() => null)
    await navigator.clipboard.writeText(payload?.joinUrl || payload?.url || meetingMeta.room)
    setWorkspaceNotice('Ссылка комнаты скопирована')
  }

  async function showRoomSettings() {
    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/settings`).catch(() => null)
    if (payload) {
      setWorkspaceNotice(`Запись ${payload.recording ? 'включена' : 'выключена'}, автоотчёт ${payload.autoReport ? 'включен' : 'выключен'}`)
    }
  }

  async function openAskAI() {
    const payload = await apiRequest('/api/ask-ai').catch(() => null)
    if (payload?.url) {
      window.open(payload.url, '_blank', 'noopener,noreferrer')
    }
  }

  async function refreshNotifications() {
    const payload = await apiRequest('/api/notifications').catch(() => null)
    if (payload) {
      setNotifications(payload)
      setWorkspaceNotice(payload.items?.[0]?.body || 'Уведомления обновлены')
    }
  }

  async function refreshProfile() {
    const payload = await apiRequest('/api/profile').catch(() => null)
    if (payload) {
      setProfile(payload)
      setWorkspaceNotice(`${payload.name} · ${payload.role || 'profile'}`)
    }
  }

  async function refreshLocales() {
    const payload = await apiRequest('/api/locales').catch(() => null)
    if (payload) {
      setLocales(payload)
      setWorkspaceNotice(`Язык: ${payload.items?.find((item) => item.id === payload.current)?.label || payload.current}`)
    }
  }

  function resetReportFilters() {
    setReportSearchText('')
    setActiveReportMode('reports')
    setTimeFilterMode('all')
    setTimeFilterRange({ from: null, to: null })
    setDraftTimeFilterRange({ from: null, to: null })
    setSelectedTypeFilterIds(typeFilterOptions.map((option) => option.id))
    setWorkspaceNotice('Фильтры сброшены')
  }

  async function openReportRecording() {
    if (!selectedReport?.id) {
      return
    }

    const payload = await apiRequest(`/api/reports/${selectedReport.id}/recording`).catch(() => null)
    if (payload) {
      setReportActionMessage(`Запись: ${payload.duration}, маркеры ${payload.markers?.join(', ') || 'нет'}`)
    }
  }

  function focusCopilotPanel() {
    setReportActionMessage('Copilot открыт и готов отвечать по отчёту')
  }

  function collapseCopilotPanel() {
    setReportActionMessage('Copilot можно свернуть на следующем шаге UI')
  }

  function openReport(reportId) {
    setOpenReportActionsId('')
    setSelectedReportId(reportId)
    setActiveReportTab('notes')
    setActiveView('reportDetail')
    window.history.replaceState(null, '', `#report/${reportId}`)
  }

  function handleReportRowKeyDown(event, reportId) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      openReport(reportId)
    }
  }

  function toggleReportActions(event, reportId) {
    event.stopPropagation()
    setOpenReportActionsId((current) => (current === reportId ? '' : reportId))
    if (!reportActions[reportId]) {
      apiRequest(`/api/reports/${reportId}/actions`)
        .then((payload) => setReportActions((current) => ({ ...current, [reportId]: payload })))
        .catch(() => {})
    }
  }

  function keepReportActionsOpen(event) {
    event.stopPropagation()
  }

  async function uploadReport() {
    setReportActionMessage('')

    try {
      const payload = await apiRequest('/api/reports/upload', {
        method: 'POST',
        body: JSON.stringify({
          title: 'Новая встреча',
          source: 'Upload',
          owner: meetingMeta.name,
          folder: 'Обработка',
        }),
      })
      const nextReport = payload.report
      if (nextReport) {
        setReports((current) => [nextReport, ...current])
        setSelectedReportId(nextReport.id)
      }
      setReportActionMessage(payload.message || 'Отчёт отправлен на обработку')
    } catch (error) {
      setReportActionMessage(error.message)
    }
  }

  async function downloadReport(reportId) {
    const response = await fetch(`/api/reports/${reportId}/download`)
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}))
      throw new Error(payload.error || 'Не удалось скачать отчёт')
    }

    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${reportId}.txt`
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  }

  async function handleReportAction(reportId, actionId) {
    setReportActionMessage('')

    try {
      if (actionId === 'download') {
        await downloadReport(reportId)
        setReportActionMessage('Скачивание отчёта началось')
      } else if (actionId === 'share') {
        const payload = await apiRequest(`/api/reports/${reportId}/share`, { method: 'POST' })
        if (navigator.clipboard) {
          await navigator.clipboard.writeText(payload.url || `/report/${reportId}`)
        }
        setReportActionMessage('Ссылка на отчёт скопирована')
      } else if (actionId === 'rename') {
        const currentTitle = (reportDetails[reportId]?.report || reports.find((report) => report.id === reportId))?.title || ''
        const title = window.prompt('Новое название отчёта', currentTitle)
        if (!title) {
          return
        }
        const payload = await apiRequest(`/api/reports/${reportId}`, {
          method: 'PATCH',
          body: JSON.stringify({ title }),
        })
        if (payload.report) {
          setReports((current) => current.map((report) => (report.id === reportId ? payload.report : report)))
          setReportDetails((current) => ({
            ...current,
            [reportId]: {
              ...(current[reportId] || {}),
              report: payload.report,
            },
          }))
        }
        setReportActionMessage('Отчёт переименован')
      } else if (actionId === 'delete') {
        await apiRequest(`/api/reports/${reportId}`, { method: 'DELETE' })
        setReports((current) => current.filter((report) => report.id !== reportId))
        setReportActionMessage('Отчёт удалён')
        if (selectedReportId === reportId) {
          const nextReport = reports.find((report) => report.id !== reportId)
          if (nextReport) {
            setSelectedReportId(nextReport.id)
          } else {
            setActiveView('reports')
          }
        }
      } else if (actionId === 'send') {
        await apiRequest(`/api/reports/${reportId}/send`, { method: 'POST' })
        setReportActionMessage('Отправка отчёта поставлена в очередь')
      } else if (actionId === 'copy') {
        const payload = await apiRequest(`/api/reports/${reportId}/copy`)
        if (navigator.clipboard) {
          await navigator.clipboard.writeText(payload.text || '')
        }
        setReportActionMessage('Текст отчёта скопирован')
      }
    } catch (error) {
      setReportActionMessage(error.message)
    } finally {
      setOpenReportActionsId('')
    }
  }

  async function runReportLookup(kind) {
    if (!selectedReportId) {
      return
    }

    const endpointByKind = {
      prompts: 'prompts',
      history: 'history',
      search: 'search?q=backend',
    }
    const endpoint = endpointByKind[kind]
    if (!endpoint) {
      return
    }

    try {
      const payload = await apiRequest(`/api/reports/${selectedReportId}/${endpoint}`)
      if (kind === 'prompts') {
        setReportActionMessage(`Prompts: ${(payload.prompts || []).length}`)
      } else if (kind === 'history') {
        setReportActionMessage(`История чата: ${(payload.history || []).length}`)
      } else {
        setReportActionMessage(`Найдено: ${(payload.results || []).length}`)
      }
    } catch (error) {
      setReportActionMessage(error.message)
    }
  }

  async function copyReportNotes() {
    if (activeReportTab !== 'notes' || !selectedReportId) {
      setReportActionMessage('Копирование заметок доступно только во вкладке Заметки')
      return
    }

    try {
      const payload = await apiRequest(`/api/reports/${selectedReportId}/notes`)
      const summary = payload.summary || selectedReportDetail?.summary || []
      const items = payload.actionItems || selectedReportDetail?.actionItems || []
      const text = [
        selectedReport.title,
        '',
        ...summary.map((section) => `${section.title}\n${section.text}`),
        '',
        'Action items',
        ...items.map((item) => `- ${item.title || item.task} (${item.owner || ''}, ${item.due || ''})`),
      ].join('\n')

      if (navigator.clipboard) {
        await navigator.clipboard.writeText(text)
      }
      setReportActionMessage('Заметки скопированы')
    } catch (error) {
      setReportActionMessage(error.message)
    }
  }

  async function editReportNotes() {
    if (!selectedReportId) {
      return
    }

    setIsDetailActionsOpen(false)

    try {
      await apiRequest(`/api/reports/${selectedReportId}/notes`, {
        method: 'PATCH',
        body: JSON.stringify({
          summary: selectedReportDetail?.summary || [],
          actionItems: selectedReportDetail?.actionItems || [],
        }),
      })
      setReportActionMessage('Заметки отправлены на редактирование')
    } catch (error) {
      setReportActionMessage(error.message || 'Backend endpoint для редактирования заметок пока недоступен')
    }
  }

  async function askReportCopilot(message) {
    const text = message.trim()
    if (!text || !selectedReportId || isCopilotSending) {
      return
    }

    setIsCopilotSending(true)
    setCopilotInput('')
    setCopilotMessages((current) => [...current, { role: 'user', text }])

    try {
      const payload = await apiRequest(`/api/reports/${selectedReportId}/chat`, {
        method: 'POST',
        body: JSON.stringify({ message: text }),
      })
      setCopilotMessages((current) => [...current, { role: 'assistant', text: payload.answer || 'Ответ пустой' }])
    } catch (error) {
      setCopilotMessages((current) => [...current, { role: 'assistant', text: error.message }])
    } finally {
      setIsCopilotSending(false)
    }
  }

  function switchView(view) {
    setActiveView(view)
    if (typeof window !== 'undefined') {
      window.history.replaceState(null, '', view === 'reports' ? '#reports' : '#meeting')
    }
  }

  function selectQuickDateOption(option) {
    const nextRange = getQuickDateRange(option)

    setTimeFilterMode(option.id)
    setTimeFilterRange(nextRange)
    setDraftTimeFilterRange(nextRange)
  }

  function selectCalendarDate(day) {
    if (day.isDisabled) {
      return
    }

    const nextDate = day.date

    setTimeFilterMode('custom')
    setDraftTimeFilterRange((current) => {
      if (!current.from || current.to) {
        return { from: nextDate, to: null }
      }

      if (nextDate < current.from) {
        return { from: nextDate, to: current.from }
      }

      return { from: current.from, to: nextDate }
    })
  }

  function applyCustomDateRange() {
    if (!draftTimeFilterRange.from) {
      return
    }

    setTimeFilterMode('custom')
    setTimeFilterRange({
      from: draftTimeFilterRange.from,
      to: draftTimeFilterRange.to || draftTimeFilterRange.from,
    })
    setIsTimeFilterOpen(false)
  }

  function toggleAllTypeFilters() {
    setSelectedTypeFilterIds((current) => (
      current.length === typeFilterOptions.length ? [] : typeFilterOptions.map((option) => option.id)
    ))
  }

  function toggleTypeFilter(optionId) {
    setSelectedTypeFilterIds((current) => (
      current.includes(optionId)
        ? current.filter((id) => id !== optionId)
        : [...current, optionId]
    ))
  }

  function renderTypeFilterDropdown() {
    return (
      <div className="type-filter-dropdown">
        <button className="type-filter-option" type="button" onClick={toggleAllTypeFilters}>
          <span>Выбрать все</span>
          <span className={areAllTypeFiltersSelected ? 'type-check active' : 'type-check'}>
            {areAllTypeFiltersSelected && <Check size={16} />}
          </span>
        </button>

        {typeFilterOptions.map((option) => {
          const isSelected = selectedTypeFilterIds.includes(option.id)

          return (
            <button
              className={isSelected ? 'type-filter-option active' : 'type-filter-option'}
              type="button"
              key={option.id}
              onClick={() => toggleTypeFilter(option.id)}
            >
              <span>{option.label}</span>
              <span className={isSelected ? 'type-check active' : 'type-check'}>
                {isSelected && <Check size={16} />}
              </span>
            </button>
          )
        })}
      </div>
    )
  }

  function renderDateFilterDropdown() {
    return (
      <div className="date-filter-dropdown">
        <div className="date-quick-list">
          {dateFilterOptions.map((option) => (
            <button
              className={timeFilterMode === option.id ? 'date-quick-option active' : 'date-quick-option'}
              type="button"
              key={option.id}
              onClick={() => selectQuickDateOption(option)}
            >
              <span>{option.label}</span>
              {timeFilterMode === option.id && <Check size={21} />}
            </button>
          ))}
        </div>

        <div className="date-calendar-panel">
          <div className="calendar-title">{timeFilterMode === 'custom' ? formatDateRange(draftTimeFilterRange) : timeFilterLabel}</div>
          <div className="calendar-nav">
            <button className="calendar-nav-button" type="button" onClick={() => setCalendarMonth((current) => shiftMonth(current, -1))} aria-label="Previous month">
              <ChevronLeft size={18} />
            </button>
            <strong>{calendarMonthNames[calendarMonth.getMonth()]} {calendarMonth.getFullYear()} г.</strong>
            <button className="calendar-nav-button" type="button" onClick={() => setCalendarMonth((current) => shiftMonth(current, 1))} aria-label="Next month">
              <ChevronRight size={18} />
            </button>
          </div>

          <div className="calendar-grid calendar-weekdays">
            {calendarWeekdays.map((day) => (
              <span key={day}>{day}</span>
            ))}
          </div>

          <div className="calendar-grid calendar-days">
            {calendarDays.map((day) => {
              const rangeStart = draftTimeFilterRange.from
              const rangeEnd = draftTimeFilterRange.to || draftTimeFilterRange.from
              const isSelected = isSameDate(day.date, rangeStart) || isSameDate(day.date, rangeEnd)
              const isInRange = isBetweenDates(day.date, rangeStart, rangeEnd)
              const dayClassName = [
                'calendar-day',
                !day.isCurrentMonth ? 'outside' : '',
                day.isDisabled ? 'disabled' : '',
                isSelected ? 'selected' : '',
                isInRange ? 'in-range' : '',
                isSameDate(day.date, reportCalendarToday) ? 'today' : '',
              ].filter(Boolean).join(' ')

              return (
                <button
                  className={dayClassName}
                  type="button"
                  key={day.key}
                  onClick={() => selectCalendarDate(day)}
                  disabled={day.isDisabled}
                >
                  {day.date.getDate()}
                </button>
              )
            })}
          </div>

          <div className="calendar-actions">
            <button className="calendar-ok-button" type="button" onClick={applyCustomDateRange} disabled={!draftTimeFilterRange.from}>
              OK
            </button>
          </div>
        </div>
      </div>
    )
  }

  function renderTopbar() {
    return (
      <header className="topbar">
        <a className="brand" href="/">
          <span className="brand-mark">
            <Sparkles size={18} />
          </span>
          <span>Alem Workspace</span>
        </a>

        <nav className="workspace-nav" aria-label="Workspace navigation">
          {navItems.map(({ id, label, icon: Icon }) => (
            <button
              className={activeView === id || (activeView === 'reportDetail' && id === 'reports') ? 'nav-item active' : 'nav-item'}
              type="button"
              key={id}
              onClick={() => switchView(id)}
            >
              <Icon size={19} strokeWidth={2} />
              <span>{label}</span>
            </button>
          ))}
        </nav>

        <div className="profile-tools">
          <button className={notifications.unread ? 'icon-button has-dot' : 'icon-button'} type="button" onClick={refreshNotifications} aria-label="Notifications" title={notifications.items?.[0]?.title || 'Notifications'}>
            <Bell size={21} />
          </button>
          <button className="profile-button" type="button" onClick={refreshProfile}>
            <span className="avatar">{meetingMeta.initial}</span>
            <span>{meetingMeta.name}</span>
            <ChevronDown size={17} />
          </button>
        </div>
      </header>
    )
  }

  function renderMeetingView() {
    const meetingToolbar = (
      <div className="meeting-toolbar">
        <div className="room-summary">
          <span className="live-pill">
            <Radio size={16} />
            {isConnected ? 'LIVE' : 'READY'}
          </span>
          <div>
            <h2>{meetingMeta.room}</h2>
            <p>{isConnected ? 'LiveKit conference запущена' : 'Ожидает подключения'}</p>
          </div>
        </div>

        <div className="meeting-actions">
          <button className="icon-button" type="button" onClick={copyRoom} aria-label="Copy room name">
            <Copy size={18} />
          </button>
          <button className="icon-button" type="button" onClick={copyRoom} aria-label="Room link">
            <Link size={18} />
          </button>
          <button className="icon-button" type="button" onClick={showRoomSettings} aria-label="Meeting settings">
            <Settings size={18} />
          </button>
          {isConnected && (
            <button className="danger-action" type="button" onClick={leaveMeeting}>
              Завершить
            </button>
          )}
        </div>
      </div>
    )

    return (
      <>
        <section className="meeting-hero" aria-labelledby="meeting-title">
          <div>
            <span className="date-label">AlemLive</span>
            <h1 id="meeting-title">Комната для созвона</h1>
            <p>Создайте новую комнату одним нажатием или подключитесь по названию уже существующей комнаты.</p>
          </div>

          <div className="hero-actions">
            <button
              className="primary-action"
              type="button"
              onClick={() => {
                selectEntryMode('create')
                startMeeting('create')
              }}
              disabled={isStarting}
            >
              {isStarting && entryMode === 'create' ? <Loader2 className="spin-icon" size={20} /> : <Sparkles size={20} />}
              Создать комнату
            </button>
            <button className="soft-action" type="button" onClick={() => selectEntryMode('join')}>
              <Link size={18} />
              Присоединиться
            </button>
          </div>
        </section>

        <section className="meeting-grid" aria-label="Meeting workspace">
          <aside className="control-column">
            <section className="panel join-panel">
              <div className="panel-heading">
                <span className="panel-icon">
                  <Video size={22} />
                </span>
                <div>
                  <h2>LiveKit meeting</h2>
                  <p>{isConnected ? 'Комната активна' : 'URL и token будут получены автоматически'}</p>
                </div>
              </div>

              <form className="join-form" onSubmit={joinMeeting}>
                <div className="entry-switch" aria-label="Выберите действие">
                  <button
                    className={entryMode === 'create' ? 'entry-option active' : 'entry-option'}
                    type="button"
                    onClick={() => selectEntryMode('create')}
                  >
                    <Sparkles size={18} />
                    Создать комнату
                  </button>
                  <button
                    className={entryMode === 'join' ? 'entry-option active' : 'entry-option'}
                    type="button"
                    onClick={() => selectEntryMode('join')}
                  >
                    <Link size={18} />
                    Присоединиться
                  </button>
                </div>

                <label>
                  <span>Ваше имя</span>
                  <input name="userName" value={form.userName} onChange={updateField} autoComplete="name" />
                </label>

                <label>
                  <span>Название комнаты</span>
                  <input
                    name="roomName"
                    value={form.roomName}
                    onChange={updateField}
                    placeholder="alem-meeting"
                    autoComplete="off"
                  />
                </label>

                {joinError && <p className="form-error">{joinError}</p>}

                <button className="join-button" type="submit" disabled={!canStart || isStarting}>
                  {isStarting ? <Loader2 className="spin-icon" size={18} /> : <Play size={18} fill="currentColor" />}
                  {isConnected
                    ? 'Переподключиться'
                    : entryMode === 'create'
                      ? 'Создать и войти'
                      : 'Войти по названию'}
                </button>
              </form>
            </section>

            {!isConnected && (
              <section className="panel compact-panel">
                <div className="panel-heading">
                  <span className="panel-icon">
                    <ShieldCheck size={21} />
                  </span>
                  <div>
                    <h2>Перед входом</h2>
                    <p>Выберите, что включить сразу в комнате</p>
                  </div>
                </div>

                <div className="device-toggles">
                  <button
                    className={devices.mic ? 'device-toggle active' : 'device-toggle'}
                    type="button"
                    onClick={() => toggleDevice('mic')}
                    aria-pressed={devices.mic}
                  >
                    {devices.mic ? <Mic size={19} /> : <MicOff size={19} />}
                    <span>Микрофон</span>
                    <strong>{devices.mic ? 'включен' : 'выключен'}</strong>
                  </button>
                  <button
                    className={devices.camera ? 'device-toggle active' : 'device-toggle'}
                    type="button"
                    onClick={() => toggleDevice('camera')}
                    aria-pressed={devices.camera}
                  >
                    {devices.camera ? <Video size={19} /> : <CameraOff size={19} />}
                    <span>Камера</span>
                    <strong>{devices.camera ? 'включена' : 'выключена'}</strong>
                  </button>
                </div>
              </section>
            )}
          </aside>

          {isConnected ? (
            <LiveKitRoom
              serverUrl={meeting.serverUrl}
              token={meeting.token}
              connect
              audio={meeting.audio}
              video={meeting.video}
              onDisconnected={leaveMeeting}
              data-lk-theme="default"
              className="livekit-context"
            >
              <section className="meeting-panel">
                {meetingToolbar}

                <div className="livekit-stage connected">
                  <VideoConference />
                </div>
              </section>

              <aside className="side-column">
                <ParticipantsPanel />
              </aside>
            </LiveKitRoom>
          ) : (
            <>
              <section className="meeting-panel">
                {meetingToolbar}

                <div className="livekit-stage">
                  <div className="empty-meeting">
                    <div className="empty-orbit">
                      <Bot size={34} />
                    </div>
                    <h3>{entryMode === 'create' ? 'Готово к созданию комнаты' : 'Готово к подключению'}</h3>
                    <p>
                      {entryMode === 'create'
                        ? 'Нажмите создать комнату, и приложение само получит LiveKit token через backend.'
                        : 'Введите название комнаты и войдите. URL и token вводить вручную больше не нужно.'}
                    </p>
                  </div>
                </div>
              </section>

              <aside className="side-column">
                <ParticipantsList participants={[]} />
              </aside>
            </>
          )}
        </section>
      </>
    )
  }

  function renderReportsList() {
    return (
      <section className="reports-page">
        <div className="reports-titlebar">
          <button className="back-title-button" type="button" onClick={() => switchView('meeting')} aria-label="Back to meeting">
            <ArrowLeft size={24} />
          </button>
          <h1>Отчёты</h1>
        </div>

        <div className="ask-read-bar">
          <button className="ask-locale" type="button" onClick={refreshLocales}>
            <Grid2X2 size={18} />
            <span>{locales.current?.toUpperCase?.() || 'RU'}</span>
            <ChevronDown size={16} />
          </button>
          <span>Спросите Alem о чём угодно...</span>
          <button className="ask-send" type="button" onClick={openAskAI} aria-label="Send question">
            <Send size={19} />
          </button>
        </div>

        <div className="reports-subnav">
          <div className="report-mode-tabs">
            <button className={activeReportMode === 'reports' ? 'report-mode active' : 'report-mode'} type="button" onClick={() => setActiveReportMode('reports')}>Отчёты</button>
            <button className={activeReportMode === 'incomplete' ? 'report-mode active' : 'report-mode'} type="button" onClick={() => setActiveReportMode('incomplete')}>Неполный</button>
          </div>
          <div className="last-updated">
            <RefreshCw size={16} />
            Последнее обновление в 15:37
          </div>
          <button className="primary-action upload-action" type="button" onClick={uploadReport}>
            <Download size={18} />
            Загрузить
          </button>
        </div>

        <div className="reports-filters">
          <label className="report-search-filter">
            <Search size={18} />
            <input
              value={reportSearchText}
              onChange={(event) => setReportSearchText(event.target.value)}
              placeholder="Фильтр по названию отчёта"
            />
          </label>
          <button className="filter-button" type="button" onClick={resetReportFilters}>
            <FileText size={17} />
            Все отчёты
            <ChevronDown size={16} />
          </button>
          <div className="time-filter-wrap">
            <button
              className={isTimeFilterOpen ? 'filter-button time-filter-button active' : 'filter-button time-filter-button'}
              type="button"
              onClick={() => {
                setIsTimeFilterOpen((current) => !current)
                setIsTypeFilterOpen(false)
              }}
              aria-expanded={isTimeFilterOpen}
            >
              <CalendarDays size={17} />
              {timeFilterLabel}
              <ChevronDown size={16} />
            </button>
            {isTimeFilterOpen && renderDateFilterDropdown()}
          </div>
          <div className="type-filter-wrap">
            <button
              className={isTypeFilterOpen ? 'filter-button type-filter-button active' : 'filter-button type-filter-button'}
              type="button"
              onClick={() => {
                setIsTypeFilterOpen((current) => !current)
                setIsTimeFilterOpen(false)
              }}
              aria-expanded={isTypeFilterOpen}
            >
              <Filter size={17} />
              Тип
              <ChevronDown size={16} />
            </button>
            {isTypeFilterOpen && renderTypeFilterDropdown()}
          </div>
        </div>

        {(workspaceNotice || reportsError || reportActionMessage) && (
          <div className={reportsError ? 'reports-status error' : 'reports-status'}>
            {reportsError || reportActionMessage || workspaceNotice}
          </div>
        )}

        <div className="reports-table">
          <div className="reports-table-head">
            <span>Источник</span>
            <span>Отчёт</span>
            <span>
              Дата и время
              <ArrowDown size={17} />
            </span>
            <span>Папки</span>
            <span>Владелец</span>
          </div>

          <div className="reports-week">{reportsLoading ? 'ЗАГРУЗКА ОТЧЁТОВ...' : visibleReports[0]?.week || 'ОТЧЁТЫ'}</div>

          {visibleReports.map((report, index) => (
            <article
              className={index === 0 ? 'report-row selected' : 'report-row'}
              key={report.id}
              role="button"
              tabIndex={0}
              onClick={() => openReport(report.id)}
              onKeyDown={(event) => handleReportRowKeyDown(event, report.id)}
            >
              <span className={`report-thumb ${report.thumbnailTone}`}>
                <Video size={17} />
              </span>

              <span className="report-name-cell">
                <strong>{report.title}</strong>
                <span>
                  <small>
                    <Users size={13} />
                    {report.participants}
                  </small>
                  <small className="score-pill">
                    <Sparkles size={13} />
                    {report.score}
                  </small>
                </span>
              </span>

              <span className="date-cell">
                <strong>{report.date}</strong>
                <small>{report.time}</small>
              </span>

              <span className="folder-pill">
                <Folder size={15} />
                {report.folder}
                <Lock size={13} />
              </span>

              <span className="owner-cell">
                <b>{report.ownerInitial}</b>
                <span className="report-actions-wrap">
                  <button
                    className={openReportActionsId === report.id ? 'report-actions-button active' : 'report-actions-button'}
                    type="button"
                    onClick={(event) => toggleReportActions(event, report.id)}
                    aria-label={`Действия для отчёта ${report.title}`}
                    aria-expanded={openReportActionsId === report.id}
                  >
                    <MoreHorizontal size={22} />
                  </button>
                  {openReportActionsId === report.id && (
                    <span className="report-actions-menu" onClick={keepReportActionsOpen}>
                      {(reportActions[report.id] || [
                        { id: 'share', label: 'Поделиться', enabled: true },
                        { id: 'download', label: 'Скачать', enabled: true },
                        { id: 'rename', label: 'Переименовать отчет', enabled: true },
                        { id: 'delete', label: 'Удалить отчет', enabled: true, danger: true },
                      ]).map((action) => (
                        <button
                          className={action.danger ? 'report-action-item danger' : action.enabled === false ? 'report-action-item disabled' : 'report-action-item'}
                          type="button"
                          key={action.id}
                          disabled={action.enabled === false}
                          onClick={(event) => {
                            event.stopPropagation()
                            handleReportAction(report.id, action.id)
                          }}
                        >
                          {action.id === 'share' ? <Share2 size={19} /> : action.id === 'download' ? <Download size={19} /> : action.id === 'rename' ? <Edit3 size={19} /> : action.id === 'delete' ? <Trash2 size={19} /> : action.id === 'send' ? <Send size={19} /> : <Copy size={19} />}
                          {action.label}
                        </button>
                      ))}
                    </span>
                  )}
                </span>
              </span>
            </article>
          ))}
        </div>
      </section>
    )
  }

  function renderReportPane() {
    const detailSummary = selectedReportDetail?.summary || []
    const detailActionItems = selectedReportDetail?.actionItems || actionItems
    const detailTranscriptLines = selectedReportDetail?.transcriptLines || selectedReportDetail?.transcript || transcriptLines
    const detailSpeakerStats = selectedReportDetail?.speakerStats || speakerStats
    const detailHighlights = selectedReportDetail?.highlights || highlights
    const detailChapters = selectedReportDetail?.chapters || chapters

    if (activeReportTab === 'notes') {
      return (
        <div className="detail-notes">
          <div className="score-strip">
            <div>
              <span>Оценка Alem</span>
              <strong>{selectedReport.score}</strong>
              <small>ХОРОШО</small>
            </div>
            <div>
              <span>Вовлечённость</span>
              <strong>93</strong>
              <small>ХОРОШО</small>
            </div>
            <div>
              <span>Настроение</span>
              <strong>85</strong>
              <small>ХОРОШО</small>
            </div>
          </div>

          <section className="report-main-panel">
            <div className="section-kicker">
              <Sparkles size={18} />
              Сводка
            </div>
            <span className="edited-pill">Отредактировано</span>
            <h3>{detailSummary[0]?.title || 'Команда согласовала новый сценарий входа и структуру AI отчёта'}</h3>
            <p>
              {detailSummary[0]?.text || 'Встреча была посвящена настройке AlemLive и аналитического отчёта после созвона. Участники договорились, что пользователь должен создавать комнату и присоединяться по названию, а URL и token должны подтягиваться автоматически через backend. После встречи агент показывает резюме, задачи, транскрипт, метрики и главы.'}
            </p>
            {detailSummary.slice(1).map((section) => (
              <p key={section.title}>
                <strong>{section.title}: </strong>
                {section.text}
              </p>
            ))}
          </section>

          <section className="action-list-panel">
            <div className="section-kicker">
              <CheckCircle2 size={18} />
              Action Items
            </div>
            <div className="action-items">
              {detailActionItems.map((item) => (
                <article className="action-item" key={item.id || item.task || item.title}>
                  <span className="action-check">
                    <CheckCircle2 size={17} />
                  </span>
                  <div>
                    <h4>{item.task || item.title}</h4>
                    <p>{item.owner} · {item.due}</p>
                  </div>
                </article>
              ))}
            </div>
          </section>
        </div>
      )
    }

    if (activeReportTab === 'transcript') {
      return (
        <div className="transcript-report report-pane">
          <div className="transcript-tools">
            <div className="transcript-search">
              <Search size={18} />
              <span>Поиск по транскрипту: token, комната, отчёт</span>
            </div>
            <span className="report-badge muted">{detailTranscriptLines.length} моментов</span>
          </div>
          <div className="transcript-list">
            {detailTranscriptLines.map((line) => (
              <article className="transcript-line" key={line.id || `${line.time}-${line.speaker}`}>
                <time>{line.time}</time>
                <div>
                  <strong>{line.speaker}</strong>
                  <p>{line.text}</p>
                </div>
              </article>
            ))}
          </div>
        </div>
      )
    }

    if (activeReportTab === 'deepDive') {
      return (
        <div className="deep-report report-pane">
          <div className="metric-grid">
            <div className="metric-card">
              <TrendingUp size={20} />
              <span>Sentiment</span>
              <strong>82%</strong>
              <p>Позитивная динамика</p>
            </div>
            <div className="metric-card">
              <Zap size={20} />
              <span>Engagement</span>
              <strong>74%</strong>
              <p>Высокое участие</p>
            </div>
            <div className="metric-card">
              <Clock3 size={20} />
              <span>Interruptions</span>
              <strong>3</strong>
              <p>Низкий уровень перебиваний</p>
            </div>
          </div>
          <div className="speaker-table">
            {detailSpeakerStats.map((speaker) => (
              <article className="speaker-row" key={speaker.name}>
                <div>
                  <strong>{speaker.name}</strong>
                  <span>{speaker.sentiment} · {speaker.pace}</span>
                </div>
                <div className="talk-bar" aria-label={`${speaker.name} говорил ${speaker.talk || speaker.talkTime}% времени`}>
                  <span style={{ width: `${speaker.talk || speaker.talkTime}%` }} />
                </div>
                <b>{speaker.talk || speaker.talkTime}%</b>
              </article>
            ))}
          </div>
        </div>
      )
    }

    if (activeReportTab === 'highlights') {
      return (
        <div className="highlights-report report-pane">
          {detailHighlights.map((item) => (
            <article className="highlight-card" key={item.title}>
              <span className="highlight-time">{item.time}</span>
              <div>
                <h3>{item.title}</h3>
                <p>{item.note || item.text}</p>
              </div>
              <button className="icon-button" type="button" onClick={openReportRecording} aria-label={`Open highlight ${item.title}`}>
                <Play size={17} fill="currentColor" />
              </button>
            </article>
          ))}
        </div>
      )
    }

    return (
      <div className="chapters-report report-pane">
        {detailChapters.map((chapter, index) => (
          <article className="chapter-row" key={chapter.title}>
            <span className="chapter-index">{index + 1}</span>
            <time>{chapter.time || chapter.start}</time>
            <div>
              <h3>{chapter.title}</h3>
              <p>{chapter.duration || chapter.text}</p>
            </div>
          </article>
        ))}
      </div>
    )
  }

  function renderReportDetail() {
    return (
      <section className="report-detail-page">
        <div className="report-detail-header">
          <div className="detail-title-group">
            <button className="back-title-button" type="button" onClick={() => switchView('reports')} aria-label="Back to reports">
              <ArrowLeft size={24} />
            </button>
            <div>
              <h1>{selectedReport.title}</h1>
              <div className="detail-meta">
                <span>
                  <CalendarDays size={17} />
                  {selectedReport.date}
                </span>
                <span>
                  <Clock3 size={17} />
                  {selectedReport.time}
                </span>
                <span>
                  <Video size={17} />
                  {selectedReport.source}
                </span>
                <span>
                  <Users size={17} />
                  {selectedReport.participantNames || 'Alison Barker, Мади, Айдана, +1 больше'}
                </span>
              </div>
            </div>
          </div>

          <div className="detail-actions">
            <button className="soft-action" type="button" onClick={() => handleReportAction(selectedReport.id, 'download')}>
              <Download size={18} />
              Скачать
            </button>
            <button className="soft-action" type="button" onClick={() => handleReportAction(selectedReport.id, 'send')}>
              <Send size={18} />
              Отправить в...
            </button>
            <button className="soft-action" type="button" onClick={() => handleReportAction(selectedReport.id, 'share')}>
              <Share2 size={18} />
              Поделиться
            </button>
          </div>
        </div>

        {reportActionMessage && <div className="report-detail-status">{reportActionMessage}</div>}

        <div className="report-detail-layout">
          <div className="report-recording-column">
            <div className="video-player-mock">
              <div className="video-person">
                <span>{selectedReport.ownerInitial}</span>
              </div>
              <button className="video-pop-button" type="button" onClick={openReportRecording} aria-label="Expand video">
                <ExternalLink size={19} />
              </button>
              <div className="video-controls">
                <Play size={18} fill="currentColor" />
                <span>0:00 / 8:49</span>
                <div className="video-timeline">
                  {[18, 32, 48, 65, 82].map((left) => (
                    <i style={{ left: `${left}%` }} key={left} />
                  ))}
                </div>
                <Volume2 size={18} />
                <Settings size={18} />
              </div>
            </div>

            <div className="detail-tabs">
              <div className="detail-tab-list" role="tablist" aria-label="Разделы отчёта">
                {reportTabs.map(({ id, label, icon: Icon }) => (
                  <button
                    className={activeReportTab === id ? 'detail-tab active' : 'detail-tab'}
                    type="button"
                    role="tab"
                    aria-selected={activeReportTab === id}
                    key={id}
                    onClick={() => {
                      setActiveReportTab(id)
                      setIsDetailActionsOpen(false)
                    }}
                  >
                    <Icon size={18} />
                    {label}
                  </button>
                ))}
              </div>

              <div className="detail-tab-tools">
                <button className="detail-tool-button" type="button" onClick={() => runReportLookup('search')} aria-label="Поиск по отчёту">
                  <Search size={21} />
                </button>
                <button
                  className="detail-tool-button"
                  type="button"
                  onClick={copyReportNotes}
                  disabled={activeReportTab !== 'notes'}
                  aria-label="Копировать заметки"
                >
                  <Copy size={21} />
                </button>
                <div className="detail-more-wrap">
                  <button
                    className={isDetailActionsOpen ? 'detail-tool-button active' : 'detail-tool-button'}
                    type="button"
                    onClick={() => setIsDetailActionsOpen((current) => !current)}
                    aria-label="Дополнительные действия"
                    aria-expanded={isDetailActionsOpen}
                  >
                    <MoreHorizontal size={22} />
                  </button>
                  {isDetailActionsOpen && (
                    <div className="detail-more-menu">
                      <button className="detail-more-item" type="button" onClick={editReportNotes}>
                        <Edit3 size={18} />
                        Редактировать заметки
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>

            <div className="detail-report-content">{renderReportPane()}</div>
          </div>

          <aside className="report-copilot">
            <div className="copilot-tools">
              <button className="icon-button" type="button" onClick={() => runReportLookup('prompts')} aria-label="Edit prompts">
                <Edit3 size={18} />
              </button>
              <button className="icon-button" type="button" onClick={() => runReportLookup('history')} aria-label="History">
                <Clock3 size={18} />
              </button>
              <span />
              <button className="icon-button" type="button" onClick={focusCopilotPanel} aria-label="Open side panel">
                <ExternalLink size={18} />
              </button>
              <button className="icon-button" type="button" onClick={collapseCopilotPanel} aria-label="Collapse side panel">
                <PanelRight size={18} />
              </button>
            </div>

            <div className="copilot-question-list">
              {(selectedReportDetail?.aiQuestions || aiQuestions).map((question) => (
                <button className="copilot-question" type="button" key={question} onClick={() => askReportCopilot(question)}>
                  <Sparkles size={18} />
                  {question}
                </button>
              ))}
            </div>

            {copilotMessages.length > 0 && (
              <div className="copilot-chat-log">
                {copilotMessages.slice(-4).map((message, index) => (
                  <div className={message.role === 'user' ? 'copilot-message user' : 'copilot-message'} key={`${message.role}-${index}-${message.text}`}>
                    {message.text}
                  </div>
                ))}
              </div>
            )}

            <form
              className="copilot-input"
              onSubmit={(event) => {
                event.preventDefault()
                askReportCopilot(copilotInput)
              }}
            >
              <input
                value={copilotInput}
                onChange={(event) => setCopilotInput(event.target.value)}
                placeholder="Спросите Alem о чём угодно..."
              />
              <button className="ask-send" type="submit" aria-label="Ask Alem" disabled={isCopilotSending}>
                <Send size={18} />
              </button>
            </form>
          </aside>
        </div>
      </section>
    )
  }

  return (
    <main className={activeView === 'reportDetail' ? 'workspace-shell report-shell' : 'workspace-shell'}>
      {activeView !== 'reportDetail' && renderTopbar()}
      {activeView === 'meeting' && renderMeetingView()}
      {activeView === 'reports' && renderReportsList()}
      {activeView === 'reportDetail' && renderReportDetail()}
    </main>
  )
}

export default App
