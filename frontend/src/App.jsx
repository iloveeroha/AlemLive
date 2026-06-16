import { useEffect, useMemo, useRef, useState } from 'react'
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
  Maximize2,
  MessageSquareText,
  Mic,
  MicOff,
  Minimize2,
  MoreHorizontal,
  PanelRight,
  Pause,
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
  VolumeX,
  Zap,
} from 'lucide-react'
import { LiveKitRoom, VideoConference, useChat, useLocalParticipant, useParticipants } from '@livekit/components-react'
import '@livekit/components-styles'
import { ensureCryptoRandomUUID } from './crypto-polyfill.js'
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

const reportDownloadOptions = [
  { id: 'summary', label: 'Итог встречи (.txt)', extension: 'txt' },
  { id: 'transcript', label: 'Стенограмма встречи (.txt)', extension: 'txt' },
  { id: 'trailer', label: 'Трейлер встречи (.mp4)', extension: 'mp4' },
  { id: 'highlights', label: 'Основные моменты встречи (.mp4)', extension: 'mp4' },
  { id: 'video', label: 'Видео встречи (.mp4)', extension: 'mp4' },
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
  { id: 'meetings', value: 'meeting', label: 'Отчеты о встречах', aliases: ['meeting', 'meetings', 'google meet'] },
  { id: 'uploads', value: 'upload', label: 'Загрузки', aliases: ['upload'] },
  { id: 'readout', value: 'readout', label: 'Темы Readout', aliases: ['readout'] },
  { id: 'daily', value: 'daily', label: 'Ежедневные обзоры', aliases: ['daily'] },
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

function parseDurationToSeconds(value) {
  const parts = String(value || '')
    .split(':')
    .map((part) => Number.parseInt(part, 10))
    .filter((part) => Number.isFinite(part))

  if (parts.length === 3) {
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  }
  if (parts.length === 2) {
    return parts[0] * 60 + parts[1]
  }
  return 529
}

function formatPlaybackTime(seconds) {
  const safeSeconds = Math.max(0, Math.floor(seconds || 0))
  const minutes = Math.floor(safeSeconds / 60)
  const remainder = String(safeSeconds % 60).padStart(2, '0')
  return `${minutes}:${remainder}`
}

function getReportLocalDate(report) {
  if (!report?.createdAt) {
    return report?.date || ''
  }

  const date = new Date(report.createdAt)
  if (Number.isNaN(date.getTime())) {
    return report?.date || ''
  }

  return date.toLocaleDateString('ru-RU', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function getReportLocalTime(report) {
  if (!report?.createdAt) {
    return report?.time || ''
  }

  const date = new Date(report.createdAt)
  if (Number.isNaN(date.getTime())) {
    return report?.time || ''
  }

  return date.toLocaleTimeString('ru-RU', {
    hour: '2-digit',
    minute: '2-digit',
  })
}

function reportProcessingLabel(report) {
  const state = report?.processingState || report?.status || ''
  if (state === 'processing') {
    return 'Обрабатывается'
  }
  if (state === 'recording') {
    return 'Запись'
  }
  if (state === 'failed') {
    return 'Ошибка'
  }
  return ''
}

const authSessionKey = 'alemlive-auth-session-v2'
const authVerifierKey = 'alemlive-auth-verifier'
let currentAccessToken = ''

function setCurrentAccessToken(token) {
  currentAccessToken = token || ''
}

function getCurrentAccessToken() {
  if (currentAccessToken) {
    return currentAccessToken
  }

  const session = loadAuthSession()
  if (session?.accessToken) {
    setCurrentAccessToken(session.accessToken)
    return session.accessToken
  }

  return ''
}

function getAuthHeaders() {
  const token = getCurrentAccessToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function apiRequest(path, options = {}) {
  const { requireAuth = false, ...requestOptions } = options
  const authHeaders = getAuthHeaders()
  if (requireAuth && !authHeaders.Authorization) {
    throw new Error('Сессия Keycloak не найдена. Войдите заново и повторите загрузку.')
  }

  const isFormDataBody = typeof FormData !== 'undefined' && options.body instanceof FormData
  const response = await fetch(path, {
    ...requestOptions,
    headers: {
      ...(!isFormDataBody && requestOptions.body ? { 'Content-Type': 'application/json' } : {}),
      ...authHeaders,
      ...requestOptions.headers,
    },
  })

  const contentType = response.headers.get('content-type') || ''
  const payload = contentType.includes('application/json') ? await response.json().catch(() => ({})) : await response.text()

  if (!response.ok) {
    if (response.status === 401 && typeof window !== 'undefined') {
      clearAuthSession()
      window.dispatchEvent(new CustomEvent('alemlive-auth-expired', {
        detail: payload?.error === 'Missing bearer token'
          ? 'Сессия Keycloak не найдена. Войдите заново.'
          : payload?.error || payload?.message || 'Сессия истекла. Войдите заново.',
      }))
    }
    const fallbackMessage = response.status === 413
      ? 'Файл слишком большой для загрузки'
      : `Backend request failed (${response.status})`
    const textMessage = typeof payload === 'string' ? payload.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim() : ''
    throw new Error(payload?.error || payload?.message || textMessage || fallbackMessage)
  }

  return payload
}

function saveDownload(blob, filename) {
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

function loadAuthSession() {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const session = JSON.parse(window.localStorage.getItem(authSessionKey))
    if (!session?.accessToken || isJWTExpired(session.accessToken)) {
      window.localStorage.removeItem(authSessionKey)
      return null
    }
    return session
  } catch {
    window.localStorage.removeItem(authSessionKey)
    return null
  }
}

function saveAuthSession(session) {
  window.localStorage.setItem(authSessionKey, JSON.stringify(session))
  setCurrentAccessToken(session?.accessToken || '')
}

function clearAuthSession() {
  window.localStorage.removeItem(authSessionKey)
  window.sessionStorage.removeItem(authVerifierKey)
  setCurrentAccessToken('')
}

function decodeJWTClaims(token) {
  try {
    const [, payload] = token.split('.')
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(payload.length / 4) * 4, '=')
    const json = decodeURIComponent(
      atob(normalized)
        .split('')
        .map((char) => `%${char.charCodeAt(0).toString(16).padStart(2, '0')}`)
        .join(''),
    )
    return JSON.parse(json)
  } catch {
    return {}
  }
}

function isJWTExpired(token) {
  const claims = decodeJWTClaims(token)
  return !claims.exp || claims.exp * 1000 <= Date.now() + 30000
}

function base64URL(bytes) {
  const raw = String.fromCharCode(...new Uint8Array(bytes))
  return btoa(raw).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

function randomPKCEValue(length = 64) {
  const bytes = new Uint8Array(length)
  window.crypto.getRandomValues(bytes)
  return base64URL(bytes)
}

async function sha256Base64URL(value) {
  const digest = await window.crypto.subtle.digest('SHA-256', new TextEncoder().encode(value))
  return base64URL(digest)
}

function getAuthRedirectURI() {
  return `${window.location.origin}${window.location.pathname}`
}

function cleanAuthCallbackURL() {
  const url = new URL(window.location.href)
  url.searchParams.delete('code')
  url.searchParams.delete('state')
  url.searchParams.delete('session_state')
  url.searchParams.delete('iss')
  url.searchParams.delete('error')
  url.searchParams.delete('error_description')
  window.history.replaceState(null, '', `${url.pathname}${url.search}${url.hash || '#meeting'}`)
}

function getSelectedTypeValues(selectedTypeIds) {
  return typeFilterOptions
    .filter((option) => selectedTypeIds.includes(option.id))
    .map((option) => option.value)
}

function getReportTypeValue(report) {
  const rawType = report.type || report.kind || report.reportType || report.category || report.source || ''
  const normalizedType = String(rawType).trim().toLowerCase()

  if (normalizedType === 'upload') {
    return 'upload'
  }

  const matchedOption = typeFilterOptions.find((option) => option.aliases.includes(normalizedType))
  return matchedOption?.value || normalizedType
}

function filterReportsByType(rows, selectedTypeIds) {
  if (selectedTypeIds.length === typeFilterOptions.length) {
    return rows
  }

  if (!selectedTypeIds.length) {
    return []
  }

  const selectedTypes = new Set(getSelectedTypeValues(selectedTypeIds))
  return rows.filter((report) => selectedTypes.has(getReportTypeValue(report)))
}

function buildReportsQuery({ search, mode, timeFilterMode, timeFilterRange, typeIds }) {
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

  if (typeIds.length > 0 && typeIds.length < typeFilterOptions.length) {
    params.set('types', getSelectedTypeValues(typeIds).join(','))
  }

  return params.toString()
}

function getInitialReportId() {
  if (typeof window === 'undefined') {
    return ''
  }

  return getHashRoute(window.location.hash).reportId
}

function getInitialView(initialReportId) {
  if (initialReportId) {
    return 'reportDetail'
  }

  return typeof window === 'undefined' ? 'reports' : getHashRoute(window.location.hash).view
}

function getHashRoute(hash) {
  const normalizedHash = String(hash || '').trim()
  const [, reportId] = normalizedHash.match(/^#report\/(.+)$/) || []
  if (reportId) {
    return { view: 'reportDetail', reportId }
  }

  if (normalizedHash === '#meeting') {
    return { view: 'meeting', reportId: '' }
  }

  return { view: 'reports', reportId: '' }
}

function getInitialRoomName() {
  if (typeof window === 'undefined') {
    return import.meta.env.VITE_LIVEKIT_ROOM ?? 'alem-meeting'
  }

  const roomFromURL = new URLSearchParams(window.location.search).get('room')
  return roomFromURL || import.meta.env.VITE_LIVEKIT_ROOM || 'alem-meeting'
}

function getMeetingShareURL(roomName) {
  if (typeof window === 'undefined') {
    return `/?room=${encodeURIComponent(roomName)}#meeting`
  }

  const isLocalhost = ['localhost', '127.0.0.1', '::1'].includes(window.location.hostname)
  const needsHTTPS = window.location.protocol !== 'https:' && !isLocalhost
  const protocol = needsHTTPS ? 'https:' : window.location.protocol
  const port = needsHTTPS && window.location.port === '5173' ? '5174' : window.location.port
  const host = port ? `${window.location.hostname}:${port}` : window.location.hostname
  return `${protocol}//${host}/?room=${encodeURIComponent(roomName)}#meeting`
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

function getMediaErrorMessage(error) {
  const name = error?.name || ''
  const message = error?.message || ''

  if (name === 'NotAllowedError' || /permission|denied|not allowed/i.test(message)) {
    return 'Разрешите доступ к камере и микрофону в браузере'
  }

  if (name === 'NotFoundError' || /not found|device not found/i.test(message)) {
    return 'Камера или микрофон не найдены'
  }

  if (name === 'NotReadableError' || /busy|in use|could not start/i.test(message)) {
    return 'Камера или микрофон заняты другим приложением'
  }

  if (shouldWarnAboutMediaSecurity()) {
    return 'Откройте встречу через HTTPS или localhost, иначе браузер может блокировать камеру и микрофон'
  }

  return message || 'Не удалось включить камеру или микрофон'
}

function shouldWarnAboutMediaSecurity() {
  if (typeof window === 'undefined') {
    return false
  }

  return window.location.protocol !== 'https:' && !['localhost', '127.0.0.1', '::1'].includes(window.location.hostname)
}

function LiveKitDeviceButtons({ onDeviceStateChange, onDevicePreferenceChange, onDeviceError }) {
  const { localParticipant, isMicrophoneEnabled, isCameraEnabled, lastMicrophoneError, lastCameraError } = useLocalParticipant()
  const [pendingDevice, setPendingDevice] = useState('')

  useEffect(() => {
    onDeviceStateChange('mic', isMicrophoneEnabled)
  }, [isMicrophoneEnabled, onDeviceStateChange])

  useEffect(() => {
    onDeviceStateChange('camera', isCameraEnabled)
  }, [isCameraEnabled, onDeviceStateChange])

  useEffect(() => {
    if (lastMicrophoneError) {
      onDeviceError('mic', lastMicrophoneError)
    }
  }, [lastMicrophoneError, onDeviceError])

  useEffect(() => {
    if (lastCameraError) {
      onDeviceError('camera', lastCameraError)
    }
  }, [lastCameraError, onDeviceError])

  async function toggleLiveKitDevice(name) {
    if (!localParticipant || pendingDevice) {
      return
    }

    const nextEnabled = name === 'mic' ? !isMicrophoneEnabled : !isCameraEnabled

    try {
      setPendingDevice(name)
      if (name === 'mic') {
        await localParticipant.setMicrophoneEnabled(nextEnabled)
      } else {
        await localParticipant.setCameraEnabled(nextEnabled)
      }

      onDevicePreferenceChange(name, nextEnabled)
    } catch (error) {
      onDeviceError(name, error)
    } finally {
      setPendingDevice('')
    }
  }

  return (
    <>
      <button
        className={isMicrophoneEnabled ? 'icon-button active' : 'icon-button'}
        type="button"
        onClick={() => toggleLiveKitDevice('mic')}
        disabled={pendingDevice === 'mic'}
        aria-label={isMicrophoneEnabled ? 'Выключить микрофон' : 'Включить микрофон'}
        aria-pressed={isMicrophoneEnabled}
      >
        {isMicrophoneEnabled ? <Mic size={18} /> : <MicOff size={18} />}
      </button>
      <button
        className={isCameraEnabled ? 'icon-button active' : 'icon-button'}
        type="button"
        onClick={() => toggleLiveKitDevice('camera')}
        disabled={pendingDevice === 'camera'}
        aria-label={isCameraEnabled ? 'Выключить камеру' : 'Включить камеру'}
        aria-pressed={isCameraEnabled}
      >
        {isCameraEnabled ? <Video size={18} /> : <CameraOff size={18} />}
      </button>
    </>
  )
}

function ConferenceChatPanel({ onClose }) {
  const { chatMessages, send, isSending } = useChat()
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  async function submitMessage(event) {
    event.preventDefault()
    const text = message.trim()
    if (!text || isSending) {
      return
    }

    try {
      setError('')
      ensureCryptoRandomUUID()
      await send(text)
      setMessage('')
    } catch (sendError) {
      setError(sendError?.message || 'Не удалось отправить сообщение')
    }
  }

  return (
    <section className="panel conference-chat-panel">
      <div className="panel-heading">
        <span className="panel-icon">
          <MessageSquareText size={21} />
        </span>
        <div>
          <h2>Чат встречи</h2>
          <p>Сообщения LiveKit</p>
        </div>
        {onClose && (
          <button className="icon-button conference-chat-close" type="button" onClick={onClose} aria-label="Скрыть чат">
            <ChevronRight size={18} />
          </button>
        )}
      </div>

      <div className="conference-chat" role="log" aria-label="Сообщения встречи">
        <div className="conference-chat-messages">
          {chatMessages.length === 0 ? (
            <p className="conference-chat-empty">Пока сообщений нет</p>
          ) : (
            chatMessages.map((item) => {
              const author = item.from?.name || item.from?.identity || 'Участник'
              const sentAt = item.timestamp ? new Date(item.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : ''

              return (
                <article className="conference-message" key={`${item.timestamp}-${author}-${item.message}`}>
                  <div>
                    <strong>{author}</strong>
                    {sentAt && <span>{sentAt}</span>}
                  </div>
                  <p>{item.message}</p>
                </article>
              )
            })
          )}
        </div>

        <form className="conference-chat-form" onSubmit={submitMessage}>
          <input
            value={message}
            onChange={(event) => setMessage(event.target.value)}
            placeholder="Написать в чат встречи..."
            aria-label="Сообщение в чат встречи"
          />
          <button className="ask-send" type="submit" disabled={isSending || !message.trim()} aria-label="Отправить сообщение">
            {isSending ? <Loader2 className="spin-icon" size={18} /> : <Send size={18} />}
          </button>
        </form>

        {error && <p className="conference-chat-error">{error}</p>}
      </div>
    </section>
  )
}

function App() {
  const initialReportId = getInitialReportId()
  const manualDisconnectRef = useRef(false)
  const [activeView, setActiveView] = useState(() => getInitialView(initialReportId))
  const [selectedReportId, setSelectedReportId] = useState(initialReportId || reportRows[0].id)
  const [activeReportTab, setActiveReportTab] = useState('notes')
  const [form, setForm] = useState({
    roomName: getInitialRoomName(),
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
  const [authConfig, setAuthConfig] = useState({ enabled: false })
  const [authSession, setAuthSession] = useState(() => loadAuthSession())
  const [authError, setAuthError] = useState('')
  const [authReady, setAuthReady] = useState(false)
  const [notifications, setNotifications] = useState({ unread: 0, items: [] })
  const [locales, setLocales] = useState({ current: 'ru', items: [] })
  const [reports, setReports] = useState(reportRows)
  const [reportFilters, setReportFilters] = useState(null)
  const [reportDetails, setReportDetails] = useState({})
  const [reportActions, setReportActions] = useState({})
  const [reportsError, setReportsError] = useState('')
  const [reportsLoading, setReportsLoading] = useState(false)
  const [reportsRefreshKey, setReportsRefreshKey] = useState(0)
  const [activeReportMode, setActiveReportMode] = useState('reports')
  const [reportSearchText, setReportSearchText] = useState('')
  const [workspaceNotice, setWorkspaceNotice] = useState('')
  const [meetingNotice, setMeetingNotice] = useState('')
  const [reportActionMessage, setReportActionMessage] = useState('')
  const [copilotInput, setCopilotInput] = useState('')
  const [copilotMessages, setCopilotMessages] = useState([])
  const [isCopilotSending, setIsCopilotSending] = useState(false)
  const [isDownloadMenuOpen, setIsDownloadMenuOpen] = useState(false)
  const [isDetailActionsOpen, setIsDetailActionsOpen] = useState(false)
  const [isMeetingMaximized, setIsMeetingMaximized] = useState(false)
  const [isConferenceChatOpen, setIsConferenceChatOpen] = useState(true)
  const [isCopilotCollapsed, setIsCopilotCollapsed] = useState(false)
  const [roomSettings, setRoomSettings] = useState(null)
  const [isRoomSettingsOpen, setIsRoomSettingsOpen] = useState(false)
  const [reportVideoPlaying, setReportVideoPlaying] = useState(false)
  const [reportVideoSeconds, setReportVideoSeconds] = useState(0)
  const [reportVideoMuted, setReportVideoMuted] = useState(false)
  const [reportVideoSpeed, setReportVideoSpeed] = useState(1)
  const copilotInputRef = useRef(null)
  const reportUploadInputRef = useRef(null)

  const canStart = form.userName.trim() && form.roomName.trim()
  const isConnected = Boolean(meeting)
  const isAuthEnabled = Boolean(authConfig.enabled)
  const isAuthenticated = !isAuthEnabled || Boolean(authSession?.accessToken)
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
  const visibleReports = filterReportsByType(reports.length ? reports : reportRows, selectedTypeFilterIds)
  const reportVideoDurationSeconds = parseDurationToSeconds(selectedReport?.duration || '08:49')
  const reportVideoProgress = Math.min(100, Math.max(0, (reportVideoSeconds / reportVideoDurationSeconds) * 100))
  const hasProcessingReports = reports.some((report) => ['processing', 'recording'].includes(report.processingState || report.status))
  const selectedReportRecordingUrl = selectedReportDetail?.recordingUrl || ''

  function syncViewFromHash() {
    if (typeof window === 'undefined') {
      return
    }

    const route = getHashRoute(window.location.hash)
    setActiveView(route.view)
    if (route.reportId) {
      setSelectedReportId(route.reportId)
    }
  }

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
    setCurrentAccessToken(authSession?.accessToken || '')
  }, [authSession])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined
    }

    window.addEventListener('hashchange', syncViewFromHash)
    return () => window.removeEventListener('hashchange', syncViewFromHash)
  }, [])

  useEffect(() => {
    let isMounted = true

    async function initializeAuth() {
      try {
        const configPayload = await apiRequest('/api/auth/config')
        if (!isMounted) {
          return
        }

        setAuthConfig(configPayload)

        if (!configPayload.enabled) {
          setAuthReady(true)
          return
        }

        const callbackURL = new URL(window.location.href)
        const code = callbackURL.searchParams.get('code')
        const state = callbackURL.searchParams.get('state')
        const callbackError = callbackURL.searchParams.get('error_description') || callbackURL.searchParams.get('error')
        const verifierPayload = JSON.parse(window.sessionStorage.getItem(authVerifierKey) || '{}')

        if (callbackError) {
          cleanAuthCallbackURL()
          throw new Error(callbackError)
        }

        if (code) {
          if (!state || state !== verifierPayload.state || !verifierPayload.verifier) {
            throw new Error('Invalid Keycloak login state')
          }

          const tokenResponse = await fetch(configPayload.tokenEndpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              code,
              redirectUri: getAuthRedirectURI(),
              codeVerifier: verifierPayload.verifier,
            }),
          })
          const tokenPayload = await tokenResponse.json().catch(() => ({}))
          if (!tokenResponse.ok) {
            throw new Error(tokenPayload.error_description || tokenPayload.error || 'Keycloak token exchange failed')
          }

          const nextSession = {
            accessToken: tokenPayload.access_token,
            idToken: tokenPayload.id_token,
            refreshToken: tokenPayload.refresh_token,
            expiresAt: Date.now() + Number(tokenPayload.expires_in || 0) * 1000,
          }
          saveAuthSession(nextSession)
          setAuthSession(nextSession)
          cleanAuthCallbackURL()
          syncViewFromHash()
          setAuthReady(true)
          return
        }

        const storedSession = loadAuthSession()
        if (storedSession) {
          setAuthSession(storedSession)
          setCurrentAccessToken(storedSession.accessToken)
        } else {
          clearAuthSession()
          setAuthSession(null)
        }
      } catch (error) {
        clearAuthSession()
        setAuthSession(null)
        setAuthError(error.message)
      } finally {
        if (isMounted) {
          setAuthReady(true)
        }
      }
    }

    initializeAuth()

    return () => {
      isMounted = false
    }
  }, [])

  async function loginWithKeycloak() {
    clearAuthSession()
    setAuthSession(null)
    setAuthError('')
    if (!authConfig.enabled || !authConfig.authorizationEndpoint || !authConfig.clientId) {
      setAuthError('Keycloak is not configured')
      return
    }

    const verifier = randomPKCEValue()
    const state = randomPKCEValue(24)
    const challenge = await sha256Base64URL(verifier)
    window.sessionStorage.setItem(authVerifierKey, JSON.stringify({ verifier, state }))

    const authURL = new URL(authConfig.authorizationEndpoint)
    authURL.searchParams.set('client_id', authConfig.clientId)
    authURL.searchParams.set('response_type', 'code')
    authURL.searchParams.set('scope', 'openid profile email')
    authURL.searchParams.set('redirect_uri', getAuthRedirectURI())
    authURL.searchParams.set('code_challenge', challenge)
    authURL.searchParams.set('code_challenge_method', 'S256')
    authURL.searchParams.set('state', state)
    window.location.assign(authURL.toString())
  }

  function logoutFromKeycloak() {
    const idToken = authSession?.idToken
    clearAuthSession()
    setAuthSession(null)
    setProfile(null)

    if (authConfig.logoutEndpoint) {
      const logoutURL = new URL(authConfig.logoutEndpoint)
      logoutURL.searchParams.set('post_logout_redirect_uri', `${window.location.origin}${window.location.pathname}#meeting`)
      if (idToken) {
        logoutURL.searchParams.set('id_token_hint', idToken)
      }
      window.location.assign(logoutURL.toString())
    }
  }

  useEffect(() => {
    if (!authReady || !isAuthenticated) {
      return undefined
    }

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
  }, [authReady, isAuthenticated])

  useEffect(() => {
    if (!authReady || !isAuthenticated || activeView === 'meeting') {
      return undefined
    }

    let isMounted = true
    const query = buildReportsQuery({
      search: reportSearchText,
      mode: activeReportMode,
      timeFilterMode,
      timeFilterRange,
      typeIds: selectedTypeFilterIds,
    })

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
        if (nextReports.length && (!selectedReportId || (activeView === 'reports' && !nextReports.some((report) => report.id === selectedReportId)))) {
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
  }, [activeReportMode, activeView, authReady, isAuthenticated, reportSearchText, reportsRefreshKey, selectedReportId, selectedTypeFilterIds, timeFilterMode, timeFilterRange])

  useEffect(() => {
    if (!authReady || !isAuthenticated || activeView === 'meeting' || !selectedReportId) {
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
  }, [activeView, authReady, isAuthenticated, selectedReportId])

  useEffect(() => {
    if (!authReady || !isAuthenticated || !hasProcessingReports) {
      return undefined
    }

    const timer = window.setInterval(() => {
      setReportsRefreshKey((current) => current + 1)
      if (selectedReportId) {
        apiRequest(`/api/reports/${selectedReportId}`)
          .then((payload) => {
            setReportDetails((current) => ({ ...current, [selectedReportId]: payload }))
          })
          .catch(() => {})
      }
    }, 5000)

    return () => window.clearInterval(timer)
  }, [authReady, hasProcessingReports, isAuthenticated, selectedReportId])

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined
    }

    document.body.classList.toggle('meeting-maximized-active', isMeetingMaximized)
    return () => {
      document.body.classList.remove('meeting-maximized-active')
    }
  }, [isMeetingMaximized])

  useEffect(() => {
    setReportVideoPlaying(false)
    setReportVideoSeconds(0)
  }, [selectedReportId])

  useEffect(() => {
    if (!reportVideoPlaying) {
      return undefined
    }

    const timer = window.setInterval(() => {
      setReportVideoSeconds((current) => {
        const nextValue = current + reportVideoSpeed
        if (nextValue >= reportVideoDurationSeconds) {
          setReportVideoPlaying(false)
          return reportVideoDurationSeconds
        }
        return nextValue
      })
    }, 1000)

    return () => window.clearInterval(timer)
  }, [reportVideoDurationSeconds, reportVideoPlaying, reportVideoSpeed])

  function updateField(event) {
    const { name, value } = event.target
    setForm((current) => ({ ...current, [name]: value }))
    setJoinError('')
  }

  function selectEntryMode(mode) {
    setEntryMode(mode)
    setJoinError('')
  }

  function updateDevicePreference(name, enabled, options = {}) {
    const { notifyBackend = true } = options

    setDevices((current) => {
      if (current[name] === enabled) {
        return current
      }

      const next = { ...current, [name]: enabled }
      window.localStorage.setItem('alemlive-devices', JSON.stringify(next))
      return next
    })

    if (notifyBackend) {
      apiRequest('/api/devices', {
        method: 'POST',
        body: JSON.stringify({
          roomName: form.roomName || meetingMeta.room,
          userName: form.userName || meetingMeta.name,
          device: name,
          enabled,
        }),
      }).catch(() => {})
    }
  }

  function toggleDevice(name) {
    updateDevicePreference(name, !devices[name])
  }

  function handleLiveKitDeviceStateChange(name, enabled) {
    updateDevicePreference(name, enabled, { notifyBackend: false })
  }

  function handleLiveKitDevicePreferenceChange(name, enabled) {
    setMeetingNotice('')
    updateDevicePreference(name, enabled)
  }

  function handleLiveKitDeviceError(_name, error) {
    setMeetingNotice(getMediaErrorMessage(error))
    updateDevicePreference(_name, false, { notifyBackend: false })
  }

  function handleLiveKitError(error) {
    setMeetingNotice(getMediaErrorMessage(error))
  }

  function handleLiveKitDisconnected() {
    setMeeting(null)
    setIsMeetingMaximized(false)
    if (manualDisconnectRef.current) {
      manualDisconnectRef.current = false
      setMeetingNotice('')
      return
    }

    if (meeting?.roomName && meeting?.userName) {
      recordMeetingEvent('left')
    }
    setMeetingNotice('Соединение с комнатой разорвано')
  }

  useEffect(() => {
    function handleAuthExpired(event) {
      setAuthSession(null)
      setAuthError(event.detail || 'Сессия истекла. Войдите заново.')
      setAuthReady(true)
    }

    window.addEventListener('alemlive-auth-expired', handleAuthExpired)
    return () => window.removeEventListener('alemlive-auth-expired', handleAuthExpired)
  }, [])

  function recordMeetingEvent(event, overrides = {}) {
    return apiRequest('/api/meetings/events', {
      method: 'POST',
      body: JSON.stringify({
        roomName: overrides.roomName || meetingMeta.room,
        userName: overrides.userName || meetingMeta.name,
        event,
      }),
    }).catch(() => null)
  }

  async function requestToken(roomName, userName, isHost) {
    const response = await fetch('/api/livekit/token', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...getAuthHeaders(),
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

    manualDisconnectRef.current = false
    setMeetingNotice('')

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
      setIsMeetingMaximized(true)
      setIsConferenceChatOpen(true)
      recordMeetingEvent(mode === 'create' ? 'created' : 'joined', {
        roomName: payload.roomName || nextRoomName,
        userName: payload.userName || nextUserName,
      })
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
    manualDisconnectRef.current = true
    await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/leave`, {
      method: 'POST',
      body: JSON.stringify({
        userName: meetingMeta.name,
        event: 'left',
      }),
    }).catch(() => null)
    setMeeting(null)
    setIsMeetingMaximized(false)
    setMeetingNotice('')
  }

  async function copyRoomName() {
    if (!navigator.clipboard) {
      return
    }

    await navigator.clipboard.writeText(meetingMeta.room)
    setWorkspaceNotice('Название комнаты скопировано')
  }

  async function copyRoomLink() {
    if (!navigator.clipboard) {
      return
    }

    await navigator.clipboard.writeText(getMeetingShareURL(meetingMeta.room))
    setWorkspaceNotice('Ссылка на комнату скопирована')
  }

  async function openRoomSettings() {
    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/settings`).catch(() => null)
    if (!payload) {
      return
    }

    setRoomSettings(payload)
    setIsRoomSettingsOpen((current) => !current)
    setWorkspaceNotice(`Настройки комнаты: запись ${payload.recording ? 'включена' : 'выключена'}, автоотчёт ${payload.autoReport ? 'включён' : 'выключен'}`)
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
      setWorkspaceNotice(`${payload.name} В· ${payload.role || 'profile'}`)
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
      if (payload.url) {
        window.open(payload.url, '_blank', 'noopener,noreferrer')
      }
      setReportActionMessage(`Запись: ${payload.duration}, маркеры ${payload.markers?.join(', ') || 'нет'}`)
    }
  }

  function toggleReportVideoPlayback() {
    setReportVideoPlaying((current) => {
      if (!current && reportVideoSeconds >= reportVideoDurationSeconds) {
        setReportVideoSeconds(0)
      }
      return !current
    })
  }

  function seekReportVideo(event) {
    const rect = event.currentTarget.getBoundingClientRect()
    const ratio = rect.width ? (event.clientX - rect.left) / rect.width : 0
    const nextSeconds = Math.round(Math.min(1, Math.max(0, ratio)) * reportVideoDurationSeconds)
    setReportVideoSeconds(nextSeconds)
  }

  function toggleReportVideoMute() {
    setReportVideoMuted((current) => !current)
    setReportActionMessage(reportVideoMuted ? 'Звук видео включён' : 'Звук видео выключен')
  }

  function cycleReportVideoSpeed() {
    const speeds = [1, 1.25, 1.5, 2]
    const currentIndex = speeds.indexOf(reportVideoSpeed)
    const nextSpeed = speeds[(currentIndex + 1) % speeds.length]
    setReportVideoSpeed(nextSpeed)
    setReportActionMessage(`Скорость видео: ${nextSpeed}x`)
  }

  function focusCopilotPanel() {
    setIsCopilotCollapsed(false)
    window.setTimeout(() => copilotInputRef.current?.focus(), 0)
    setReportActionMessage('Copilot открыт и готов отвечать по отчёту')
  }

  function collapseCopilotPanel() {
    setIsCopilotCollapsed(true)
    setReportActionMessage('Copilot можно свернуть на следующем шаге UI')
  }

  function openReport(reportId) {
    setOpenReportActionsId('')
    setIsDownloadMenuOpen(false)
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

  function toggleDownloadMenu(event) {
    event.stopPropagation()
    setIsDownloadMenuOpen((current) => !current)
    setIsDetailActionsOpen(false)
  }

  function keepDownloadMenuOpen(event) {
    event.stopPropagation()
  }

  function selectDownloadOption(reportId, optionId) {
    setIsDownloadMenuOpen(false)
    handleReportAction(reportId, `download:${optionId}`)
  }

  async function uploadReport() {
    if (reportUploadInputRef.current) {
      reportUploadInputRef.current.value = ''
      reportUploadInputRef.current.click()
      return
    }

    setReportActionMessage('')

    try {
      const payload = await apiRequest('/api/reports/upload', {
        method: 'POST',
        requireAuth: true,
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
        setActiveReportMode('reports')
        setReportsRefreshKey((current) => current + 1)
      }
      setReportActionMessage(payload.message || 'Отчёт отправлен на обработку')
    } catch (error) {
      setReportActionMessage(error.message)
    }
  }

  async function uploadReportFile(event) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }

    setReportActionMessage('')

    try {
      const body = new FormData()
      body.append('file', file)
      body.append('roomName', meetingMeta.room || 'alem-meeting')
      body.append('title', file.name.replace(/\.[^.]+$/, '') || 'Uploaded meeting')
      body.append('source', 'Upload')
      body.append('owner', meetingMeta.name || 'Team AI')
      body.append('folder', 'Processed')

      const payload = await apiRequest('/api/reports/upload', {
        method: 'POST',
        requireAuth: true,
        body,
      })
      const nextReport = payload.report
      if (nextReport) {
        setReports((current) => [nextReport, ...current.filter((report) => report.id !== nextReport.id)])
        setSelectedReportId(nextReport.id)
        setActiveReportMode('reports')
        setReportsRefreshKey((current) => current + 1)
      }
      if (payload.detail && nextReport?.id) {
        setReportDetails((current) => ({ ...current, [nextReport.id]: payload.detail }))
      }
      setActiveReportTab('notes')
      setReportActionMessage(payload.message || 'Recording transcribed and analyzed')
    } catch (error) {
      setReportActionMessage(error.message)
    } finally {
      event.target.value = ''
    }
  }

  async function downloadReport(reportId, optionId = 'summary') {
    const option = reportDownloadOptions.find((item) => item.id === optionId) || reportDownloadOptions[0]
    const filename = `${reportId}-${option.id}.${option.extension}`

    const response = await fetch(`/api/reports/${reportId}/download?format=${encodeURIComponent(option.id)}`, {
      headers: getAuthHeaders(),
    })
    if (!response.ok) {
      const contentType = response.headers.get('content-type') || ''
      const payload = contentType.includes('application/json')
        ? await response.json().catch(() => ({}))
        : { error: await response.text().catch(() => '') }
      throw new Error(payload.error || payload.message || 'Не удалось скачать файл')
    }

    const blob = await response.blob()
    saveDownload(blob, filename)
    return `Скачивание: ${option.label}`
  }

  async function handleReportAction(reportId, actionId) {
    setReportActionMessage('')
    setIsDownloadMenuOpen(false)

    try {
      if (actionId === 'download' || actionId.startsWith('download:')) {
        const message = await downloadReport(reportId, actionId.split(':')[1] || 'summary')
        setReportActionMessage(message)
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

    const searchQuery = kind === 'search' ? window.prompt('Что найти в отчёте?', 'backend') : ''
    if (kind === 'search' && searchQuery === null) {
      return
    }

    const endpointByKind = {
      prompts: 'prompts',
      history: 'history',
      search: `search?q=${encodeURIComponent(searchQuery || '')}`,
    }
    const endpoint = endpointByKind[kind]
    if (!endpoint) {
      return
    }

    try {
      const payload = await apiRequest(`/api/reports/${selectedReportId}/${endpoint}`)
      if (kind === 'prompts') {
        const prompts = payload.prompts || []
        if (prompts[0]) {
          setCopilotInput(prompts[0])
          setIsCopilotCollapsed(false)
          window.setTimeout(() => copilotInputRef.current?.focus(), 0)
        }
        setReportActionMessage(`Prompt готов: ${prompts[0] || 'нет подсказок'}`)
      } else if (kind === 'history') {
        const history = payload.history || []
        if (history.length) {
          setCopilotMessages(history.map((item) => ({ role: item.role || 'assistant', text: item.content || item.text || '' })))
        }
        setIsCopilotCollapsed(false)
        setReportActionMessage(`История чата: ${history.length || copilotMessages.length}`)
      } else {
        setReportActionMessage(`Найдено: ${(payload.results || []).length}`)
      }
    } catch (error) {
      setReportActionMessage(error.message)
    }
  }

  async function copyReportNotes() {
    if (!selectedReportId) {
      return
    }

    try {
      let text = selectedReport.title

      if (activeReportTab === 'notes') {
        const payload = await apiRequest(`/api/reports/${selectedReportId}/notes`)
        const summary = payload.summary || selectedReportDetail?.summary || []
        const items = payload.actionItems || selectedReportDetail?.actionItems || []
        text = [
          selectedReport.title,
          '',
          ...summary.map((section) => `${section.title}\n${section.text}`),
          '',
          'Action items',
          ...items.map((item) => `- ${item.title || item.task} (${item.owner || ''}, ${item.due || ''})`),
        ].join('\n')
      } else if (activeReportTab === 'transcript') {
        const payload = await apiRequest(`/api/reports/${selectedReportId}/transcript`)
        const lines = payload.lines || selectedReportDetail?.transcriptLines || transcriptLines
        text = [selectedReport.title, '', ...lines.map((line) => `${line.time || ''} ${line.speaker || ''}: ${line.text || ''}`.trim())].join('\n')
      } else if (activeReportTab === 'deepDive') {
        const payload = await apiRequest(`/api/reports/${selectedReportId}/deep-dive`)
        const stats = payload.speakerStats || selectedReportDetail?.speakerStats || speakerStats
        text = [selectedReport.title, '', ...stats.map((item) => `${item.name}: ${item.talk || item.talkTime || 0}% ${item.sentiment || ''} ${item.pace || ''}`.trim())].join('\n')
      } else if (activeReportTab === 'highlights') {
        const payload = await apiRequest(`/api/reports/${selectedReportId}/highlights`)
        const items = payload.items || selectedReportDetail?.highlights || highlights
        text = [selectedReport.title, '', ...items.map((item) => `${item.time || ''} ${item.title}: ${item.note || item.text || ''}`.trim())].join('\n')
      } else {
        const payload = await apiRequest(`/api/reports/${selectedReportId}/chapters`)
        const items = payload.items || selectedReportDetail?.chapters || chapters
        text = [selectedReport.title, '', ...items.map((item) => `${item.time || item.start || ''} ${item.title}: ${item.duration || item.text || ''}`.trim())].join('\n')
      }

      if (navigator.clipboard) {
        await navigator.clipboard.writeText(text)
      }
      setReportActionMessage('Содержимое вкладки скопировано')
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
    setIsDownloadMenuOpen(false)
    setIsDetailActionsOpen(false)
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
        <button className="brand" type="button" onClick={() => switchView('reports')}>
          <span className="brand-mark">
            <Sparkles size={18} />
          </span>
          <span>Alem Workspace</span>
        </button>

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
          {isAuthEnabled && (
            authSession?.accessToken ? (
              <button className="soft-action compact-auth-action" type="button" onClick={logoutFromKeycloak}>
                <Lock size={17} />
                Logout
              </button>
            ) : (
              <button className="soft-action compact-auth-action" type="button" onClick={loginWithKeycloak}>
                <Lock size={17} />
                Login
              </button>
            )
          )}
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
    const meetingGridClassName = [
      'meeting-grid',
      isConnected ? 'meeting-grid-connected' : '',
      isMeetingMaximized ? 'meeting-grid-maximized' : '',
      isConnected && !isConferenceChatOpen ? 'meeting-chat-collapsed' : '',
    ]
      .filter(Boolean)
      .join(' ')

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
          <button className="icon-button" type="button" onClick={copyRoomName} aria-label="Copy room name">
            <Copy size={18} />
          </button>
          <button className="icon-button" type="button" onClick={copyRoomLink} aria-label="Room link">
            <Link size={18} />
          </button>
          <button className={isRoomSettingsOpen ? 'icon-button active' : 'icon-button'} type="button" onClick={openRoomSettings} aria-label="Meeting settings" aria-pressed={isRoomSettingsOpen}>
            <Settings size={18} />
          </button>
          {isConnected && (
            <button
              className={isMeetingMaximized ? 'icon-button active' : 'icon-button'}
              type="button"
              onClick={() => setIsMeetingMaximized((current) => !current)}
              aria-label={isMeetingMaximized ? 'Свернуть видеоконференцию' : 'Развернуть видеоконференцию'}
              aria-pressed={isMeetingMaximized}
            >
              {isMeetingMaximized ? <Minimize2 size={18} /> : <Maximize2 size={18} />}
            </button>
          )}
          {isConnected && (
            <button
              className={isConferenceChatOpen ? 'icon-button active' : 'icon-button'}
              type="button"
              onClick={() => setIsConferenceChatOpen((current) => !current)}
              aria-label={isConferenceChatOpen ? 'Скрыть чат встречи' : 'Показать чат встречи'}
              aria-pressed={isConferenceChatOpen}
            >
              <MessageSquareText size={18} />
            </button>
          )}
          {isConnected && (
            <LiveKitDeviceButtons
              onDeviceStateChange={handleLiveKitDeviceStateChange}
              onDevicePreferenceChange={handleLiveKitDevicePreferenceChange}
              onDeviceError={handleLiveKitDeviceError}
            />
          )}
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

        <section className={meetingGridClassName} aria-label="Meeting workspace">
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
                {shouldWarnAboutMediaSecurity() && (
                  <p className="form-warning">
                    Для камеры и микрофона откройте встречу через HTTPS или localhost.
                  </p>
                )}
                {meetingNotice && <p className="form-error">{meetingNotice}</p>}

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
              onError={handleLiveKitError}
              onDisconnected={handleLiveKitDisconnected}
              data-lk-theme="default"
              className="livekit-context"
            >
              <section className="meeting-panel">
                {meetingToolbar}
                {isRoomSettingsOpen && roomSettings && (
                  <div className="room-settings-panel">
                    <span>Запись: {roomSettings.recording ? 'включена' : 'выключена'}</span>
                    <span>Транскрипция: {roomSettings.transcription ? 'включена' : 'выключена'}</span>
                    <span>Гости: {roomSettings.allowGuests ? 'разрешены' : 'запрещены'}</span>
                    <span>Автоотчёт: {roomSettings.autoReport ? 'включён' : 'выключен'}</span>
                  </div>
                )}

                <div className="livekit-stage connected">
                  <VideoConference />
                </div>
              </section>

              <aside className="side-column">
                <ParticipantsPanel />
                <ConferenceChatPanel onClose={() => setIsConferenceChatOpen(false)} />
              </aside>
            </LiveKitRoom>
          ) : (
            <>
              <section className="meeting-panel">
                {meetingToolbar}
                {isRoomSettingsOpen && roomSettings && (
                  <div className="room-settings-panel">
                    <span>Запись: {roomSettings.recording ? 'включена' : 'выключена'}</span>
                    <span>Транскрипция: {roomSettings.transcription ? 'включена' : 'выключена'}</span>
                    <span>Гости: {roomSettings.allowGuests ? 'разрешены' : 'запрещены'}</span>
                    <span>Автоотчёт: {roomSettings.autoReport ? 'включён' : 'выключен'}</span>
                  </div>
                )}

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

          <input
            ref={reportUploadInputRef}
            type="file"
            accept="audio/*,video/*,.webm,.mp3,.mp4,.m4a,.wav,.ogg"
            onChange={uploadReportFile}
            style={{ display: 'none' }}
          />
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

          {visibleReports.map((report) => (
            <article
              className={selectedReportId === report.id ? 'report-row selected' : 'report-row'}
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
                  {reportProcessingLabel(report) && (
                    <small className={`state-pill ${report.processingState || report.status}`}>
                      {reportProcessingLabel(report)}
                    </small>
                  )}
                </span>
              </span>

              <span className="date-cell">
                <strong>{getReportLocalDate(report)}</strong>
                <small>{getReportLocalTime(report)}</small>
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
                    <p>{item.owner} В· {item.due}</p>
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
                  <span>{speaker.sentiment} В· {speaker.pace}</span>
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
                  {getReportLocalDate(selectedReport)}
                </span>
                <span>
                  <Clock3 size={17} />
                  {getReportLocalTime(selectedReport)}
                </span>
                <span>
                  <Video size={17} />
                  {selectedReport.source}
                </span>
                <span>
                  <Users size={17} />
                  {selectedReport.participantNames || 'Мади, Айдана, Елиас, +1 больше'}
                </span>
              </div>
            </div>
          </div>

          <div className="detail-actions">
            <div className="download-menu-wrap">
              <button
                className={isDownloadMenuOpen ? 'soft-action download-menu-button active' : 'soft-action download-menu-button'}
                type="button"
                onClick={toggleDownloadMenu}
                aria-haspopup="menu"
                aria-expanded={isDownloadMenuOpen}
              >
                <Download size={18} />
                Скачать
              </button>
              {isDownloadMenuOpen && (
                <div className="download-menu" role="menu" onClick={keepDownloadMenuOpen}>
                  {reportDownloadOptions.map((option) => (
                    <button className="download-menu-item" type="button" role="menuitem" key={option.id} onClick={() => selectDownloadOption(selectedReport.id, option.id)}>
                      {option.label}
                    </button>
                  ))}
                </div>
              )}
            </div>
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
            {selectedReportRecordingUrl ? (
              <div className="report-video-frame">
                <video
                  className="report-video-player"
                  key={selectedReportRecordingUrl}
                  src={selectedReportRecordingUrl}
                  controls
                  preload="metadata"
                >
                  Ваш браузер не поддерживает просмотр видео.
                </video>
                <button className="video-pop-button" type="button" onClick={openReportRecording} aria-label="Открыть видео">
                  <ExternalLink size={19} />
                </button>
              </div>
            ) : (
              <div className="video-player-mock">
                <div className="video-person">
                  <span>{selectedReport.ownerInitial}</span>
                </div>
                <button className="video-pop-button" type="button" onClick={openReportRecording} aria-label="Expand video">
                  <ExternalLink size={19} />
                </button>
                <div className="video-controls">
                  <button className="video-control-button" type="button" onClick={toggleReportVideoPlayback} aria-label={reportVideoPlaying ? 'Пауза' : 'Воспроизвести'}>
                    {reportVideoPlaying ? <Pause size={18} fill="currentColor" /> : <Play size={18} fill="currentColor" />}
                  </button>
                  <span>{formatPlaybackTime(reportVideoSeconds)} / {formatPlaybackTime(reportVideoDurationSeconds)}</span>
                  <button
                    className="video-timeline"
                    type="button"
                    onClick={seekReportVideo}
                    style={{ '--video-progress': `${reportVideoProgress}%` }}
                    aria-label="Перемотать видео"
                  >
                    {[18, 32, 48, 65, 82].map((left) => (
                      <i style={{ left: `${left}%` }} key={left} />
                    ))}
                  </button>
                  <button className="video-control-button" type="button" onClick={toggleReportVideoMute} aria-label={reportVideoMuted ? 'Включить звук' : 'Выключить звук'}>
                    {reportVideoMuted ? <VolumeX size={18} /> : <Volume2 size={18} />}
                  </button>
                  <button className="video-control-button speed-button" type="button" onClick={cycleReportVideoSpeed} aria-label="Скорость видео">
                    {reportVideoSpeed}x
                  </button>
                  <button className="video-control-button" type="button" onClick={openReportRecording} aria-label="Настройки записи">
                    <Settings size={18} />
                  </button>
                </div>
              </div>
            )}

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
                  aria-label="Скопировать текущую вкладку"
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

          <aside className={isCopilotCollapsed ? 'report-copilot collapsed' : 'report-copilot'}>
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

            {!isCopilotCollapsed && (
              <>
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
                ref={copilotInputRef}
                value={copilotInput}
                onChange={(event) => setCopilotInput(event.target.value)}
                placeholder="Спросите Alem о чём угодно..."
              />
              <button className="ask-send" type="submit" aria-label="Ask Alem" disabled={isCopilotSending}>
                <Send size={18} />
              </button>
            </form>
              </>
            )}
          </aside>
        </div>
      </section>
    )
  }

  function renderAuthGate() {
    return (
      <main className="workspace-shell auth-shell">
        <section className="auth-panel">
          <span className="brand-mark">
            <Lock size={22} />
          </span>
          <h1>AlemLive</h1>
          <p>{authReady ? 'Войдите через Keycloak, чтобы продолжить.' : 'Проверяем авторизацию...'}</p>
          {authError && <p className="auth-error">{authError}</p>}
          {authReady && (
            <button className="primary-action" type="button" onClick={loginWithKeycloak}>
              <Lock size={18} />
              Войти через Keycloak
            </button>
          )}
        </section>
      </main>
    )
  }

  if (!authReady || (isAuthEnabled && !isAuthenticated)) {
    return renderAuthGate()
  }

  return (
    <main
      className={[
        'workspace-shell',
        activeView === 'reportDetail' ? 'report-shell' : '',
        activeView === 'meeting' && isConnected ? 'meeting-fullscreen-shell' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {activeView !== 'reportDetail' && renderTopbar()}
      {activeView === 'meeting' && renderMeetingView()}
      {activeView === 'reports' && renderReportsList()}
      {activeView === 'reportDetail' && renderReportDetail()}
    </main>
  )
}

export default App
