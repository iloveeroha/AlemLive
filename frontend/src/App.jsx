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
import { LiveKitRoom, VideoConference, useChat, useLocalParticipant, useParticipants } from '@livekit/components-react'
import '@livekit/components-styles'
import { ensureCryptoRandomUUID } from './crypto-polyfill.js'
import './App.css'

const navItems = [
  { id: 'meeting', label: 'AlemLive', icon: Grid2X2 },
  { id: 'reports', label: 'РћС‚С‡С‘С‚С‹', icon: FileText },
]

const reportTabs = [
  { id: 'notes', label: 'Р—Р°РјРµС‚РєРё', icon: FileText },
  { id: 'transcript', label: 'РўСЂР°РЅСЃРєСЂРёРїС‚', icon: MessageSquareText },
  { id: 'deepDive', label: 'Р“Р»СѓР±РѕРєРѕРµ РїРѕРіСЂСѓР¶РµРЅРёРµ', icon: BarChart3 },
  { id: 'highlights', label: 'РћСЃРЅРѕРІРЅС‹Рµ РјРѕРјРµРЅС‚С‹', icon: Highlighter },
  { id: 'chapters', label: 'Р“Р»Р°РІС‹', icon: ListChecks },
]

const reportDownloadOptions = [
  { id: 'summary', label: 'РС‚РѕРі РІСЃС‚СЂРµС‡Рё (.txt)', extension: 'txt' },
  { id: 'transcript', label: 'РЎС‚РµРЅРѕРіСЂР°РјРјР° РІСЃС‚СЂРµС‡Рё (.txt)', extension: 'txt' },
  { id: 'trailer', label: 'РўСЂРµР№Р»РµСЂ РІСЃС‚СЂРµС‡Рё (.mp4)', extension: 'mp4', pending: true },
  { id: 'highlights', label: 'РћСЃРЅРѕРІРЅС‹Рµ РјРѕРјРµРЅС‚С‹ РІСЃС‚СЂРµС‡Рё (.mp4)', extension: 'mp4', pending: true },
  { id: 'video', label: 'Р’РёРґРµРѕ РІСЃС‚СЂРµС‡Рё (.mp4)', extension: 'mp4', pending: true },
]

const reportRows = [
  {
    id: 'read-intro',
    title: 'Р’РІРѕРґ РІ Alem AI - РџСЂРёРјРµСЂ РѕС‚С‡С‘С‚Р°',
    source: 'Google Meet',
    date: 'РїС‚, 2 СЏРЅРІ. 2026 Рі.',
    time: '02:00 - 03:45',
    participants: 4,
    score: 89,
    folder: 'РћР±СЂР°Р·С†С‹ РѕС‚С‡С‘С‚РѕРІ',
    owner: 'РњР°РґРё',
    ownerInitial: 'Рњ',
    thumbnailTone: 'teal',
    week: 'РќР•Р”Р•Р›РЇ РЎ 29 Р”Р•Рљ.-4 РЇРќР’., 2025',
  },
  {
    id: 'meeting-usage',
    title: 'РСЃРїРѕР»СЊР·РѕРІР°РЅРёРµ РѕС‚С‡С‘С‚Р° СЃРѕР±СЂР°РЅРёСЏ - РџСЂРёРјРµСЂ РѕС‚С‡С‘С‚Р°',
    source: 'Google Meet',
    date: 'РїС‚, 2 СЏРЅРІ. 2026 Рі.',
    time: '01:00 - 01:04',
    participants: 4,
    score: 89,
    folder: 'РћР±СЂР°Р·С†С‹ РѕС‚С‡С‘С‚РѕРІ',
    owner: 'РђР№РґР°РЅР°',
    ownerInitial: 'Рђ',
    thumbnailTone: 'blue',
    week: 'РќР•Р”Р•Р›РЇ РЎ 29 Р”Р•Рљ.-4 РЇРќР’., 2025',
  },
  {
    id: 'copilot-search',
    title: 'РСЃРїРѕР»СЊР·СѓР№С‚Рµ Copilot РґР»СЏ РїРѕРёСЃРєР° - РџСЂРёРјРµСЂ РѕС‚С‡С‘С‚Р°',
    source: 'Google Meet',
    date: 'РїС‚, 2 СЏРЅРІ. 2026 Рі.',
    time: '00:00 - 00:07',
    participants: 4,
    score: 88,
    folder: 'РћР±СЂР°Р·С†С‹ РѕС‚С‡С‘С‚РѕРІ',
    owner: 'Р•Р»РёР°СЃ',
    ownerInitial: 'Р•',
    thumbnailTone: 'violet',
    week: 'РќР•Р”Р•Р›РЇ РЎ 29 Р”Р•Рљ.-4 РЇРќР’., 2025',
  },
  {
    id: 'mobile-guide',
    title: 'Р СѓРєРѕРІРѕРґСЃС‚РІРѕ РїРѕ РёСЃРїРѕР»СЊР·РѕРІР°РЅРёСЋ РЅР°СЃС‚РѕР»СЊРЅРѕРіРѕ Рё РјРѕР±РёР»СЊРЅРѕРіРѕ РїСЂРёР»РѕР¶РµРЅРёСЏ',
    source: 'Google Meet',
    date: 'С‡С‚, 1 СЏРЅРІ. 2026 Рі.',
    time: '23:00 - 23:04',
    participants: 5,
    score: 92,
    folder: 'РћР±СЂР°Р·С†С‹ РѕС‚С‡С‘С‚РѕРІ',
    owner: 'РљРµР»СЃРё',
    ownerInitial: 'Рљ',
    thumbnailTone: 'green',
    week: 'РќР•Р”Р•Р›РЇ РЎ 29 Р”Р•Рљ.-4 РЇРќР’., 2025',
  },
  {
    id: 'real-cases',
    title: 'РСЃСЃР»РµРґСѓР№С‚Рµ СЂРµР°Р»СЊРЅС‹Рµ СЃР»СѓС‡Р°Рё РёСЃРїРѕР»СЊР·РѕРІР°РЅРёСЏ - РџСЂРёРјРµСЂ РѕС‚С‡С‘С‚Р°',
    source: 'Google Meet',
    date: 'С‡С‚, 1 СЏРЅРІ. 2026 Рі.',
    time: '22:00 - 22:08',
    participants: 4,
    score: 87,
    folder: 'РћР±СЂР°Р·С†С‹ РѕС‚С‡С‘С‚РѕРІ',
    owner: 'РЎР°СЂР°',
    ownerInitial: 'РЎ',
    thumbnailTone: 'rose',
    week: 'РќР•Р”Р•Р›РЇ РЎ 29 Р”Р•Рљ.-4 РЇРќР’., 2025',
  },
]

const actionItems = [
  { task: 'РџРѕРґРіРѕС‚РѕРІРёС‚СЊ СЃРїРёСЃРѕРє РІРѕРїСЂРѕСЃРѕРІ РґР»СЏ РґРµРјРѕ РєР»РёРµРЅС‚Р°', owner: 'РњР°РґРё РћСЂС‹СЃР±РµРє', due: 'РЎРµРіРѕРґРЅСЏ, 18:00' },
  { task: 'РџСЂРѕРІРµСЂРёС‚СЊ backend endpoint РґР»СЏ LiveKit token', owner: 'РђР№РґР°РЅР° РЎРµР№С‚', due: 'Р—Р°РІС‚СЂР°, 11:00' },
  { task: 'РћР±РЅРѕРІРёС‚СЊ UI РѕС‚С‡С‘С‚Р° РїРѕСЃР»Рµ С‚РµСЃС‚РѕРІРѕР№ РІСЃС‚СЂРµС‡Рё', owner: 'Team AI', due: 'РџРѕСЃР»Рµ СЃРѕР·РІРѕРЅР°' },
]

const transcriptLines = [
  {
    time: '00:42',
    speaker: 'РњР°РґРё',
    text: 'РќР°Рј РЅСѓР¶РЅРѕ, С‡С‚РѕР±С‹ СѓС‡Р°СЃС‚РЅРёРє РјРѕРі РІРѕР№С‚Рё РІ РєРѕРјРЅР°С‚Сѓ С‚РѕР»СЊРєРѕ РїРѕ РЅР°Р·РІР°РЅРёСЋ, Р±РµР· СЂСѓС‡РЅРѕРіРѕ token.',
  },
  {
    time: '04:18',
    speaker: 'РђР№РґР°РЅР°',
    text: 'РџРѕСЃР»Рµ РІСЃС‚СЂРµС‡Рё РѕС‚С‡С‘С‚ РґРѕР»Р¶РµРЅ Р±С‹СЃС‚СЂРѕ РїРѕРєР°Р·С‹РІР°С‚СЊ summary, Р·Р°РґР°С‡Рё Рё РїРѕР»РЅС‹Р№ РєРѕРЅС‚РµРєСЃС‚ СЂР°Р·РіРѕРІРѕСЂР°.',
  },
  {
    time: '12:05',
    speaker: 'Team AI',
    text: 'РЇ РІС‹РґРµР»СЋ РіР»Р°РІС‹, РІРѕРїСЂРѕСЃС‹ Рё РјРµСЃС‚Р°, РіРґРµ РѕР±СЃСѓР¶РґРµРЅРёРµ Р·Р°С‚СЏРЅСѓР»РѕСЃСЊ РёР»Рё Р±С‹Р»Рѕ РѕСЃРѕР±РµРЅРЅРѕ Р°РєС‚РёРІРЅС‹Рј.',
  },
]

const speakerStats = [
  { name: 'РњР°РґРё', talk: 48, sentiment: 'РџРѕР·РёС‚РёРІРЅС‹Р№', pace: '142 СЃР»РѕРІ/РјРёРЅ' },
  { name: 'РђР№РґР°РЅР°', talk: 34, sentiment: 'РќРµР№С‚СЂР°Р»СЊРЅС‹Р№', pace: '128 СЃР»РѕРІ/РјРёРЅ' },
  { name: 'Team AI', talk: 18, sentiment: 'Р¤РѕРєСѓСЃ', pace: '96 СЃР»РѕРІ/РјРёРЅ' },
]

const highlights = [
  { time: '03:20', title: 'Р РµС€РµРЅРёРµ РїРѕ РІС…РѕРґСѓ РІ РєРѕРјРЅР°С‚Сѓ', note: 'РќР°Р·РІР°РЅРёРµ РєРѕРјРЅР°С‚С‹ СЃС‚Р°РЅРѕРІРёС‚СЃСЏ РіР»Р°РІРЅС‹Рј СЃРїРѕСЃРѕР±РѕРј РїРѕРґРєР»СЋС‡РµРЅРёСЏ.' },
  { time: '17:45', title: 'Р РёСЃРє РїРѕ backend', note: 'Р•СЃР»Рё backend РЅРµ Р·Р°РїСѓС‰РµРЅ, Р°РіРµРЅС‚ РґРѕР»Р¶РµРЅ СЏРІРЅРѕ РїРѕРєР°Р·Р°С‚СЊ РѕС€РёР±РєСѓ РїРѕРґРєР»СЋС‡РµРЅРёСЏ.' },
  { time: '28:10', title: 'РЎР»РµРґСѓСЋС‰РёР№ С€Р°Рі', note: 'Р”РѕР±Р°РІРёС‚СЊ Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєРёР№ РѕС‚С‡С‘С‚ РїРѕСЃР»Рµ Р·Р°РІРµСЂС€РµРЅРёСЏ РјРёС‚РёРЅРіР°.' },
]

const chapters = [
  { time: '00:00', title: 'РЎС‚Р°СЂС‚ Рё С†РµР»СЊ РІСЃС‚СЂРµС‡Рё', duration: '4 РјРёРЅ' },
  { time: '04:01', title: 'LiveKit РІС…РѕРґ Рё РєРѕРјРЅР°С‚С‹', duration: '9 РјРёРЅ' },
  { time: '13:10', title: 'РЎС‚СЂСѓРєС‚СѓСЂР° AI РѕС‚С‡С‘С‚Р°', duration: '12 РјРёРЅ' },
  { time: '25:30', title: 'Action items Рё С„РёРЅР°Р»СЊРЅС‹Рµ СЂРµС€РµРЅРёСЏ', duration: '7 РјРёРЅ' },
]

const aiQuestions = [
  'РљР°Рє РѕС‚РєР»СЋС‡РёС‚СЊ Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєСѓСЋ РѕС‚РїСЂР°РІРєСѓ Р·Р°РјРµС‚РѕРє РІРЅРµС€РЅРёРј СѓС‡Р°СЃС‚РЅРёРєР°Рј?',
  'РљР°Рє РїСЂРѕРІРµСЂРёС‚СЊ Рё РѕС‚СЂРµРґР°РєС‚РёСЂРѕРІР°С‚СЊ Р·Р°РјРµС‚РєРё РїРµСЂРµРґ РѕС‚РїСЂР°РІРєРѕР№?',
  'РљР°РєРёРµ РїСЂР°РІР° РЅСѓР¶РЅС‹ РґР»СЏ Search Copilot?',
  'РљР°РєРёРµ Р·Р°РґР°С‡Рё РїРѕСЏРІРёР»РёСЃСЊ РїРѕСЃР»Рµ РІСЃС‚СЂРµС‡Рё?',
  'РџРµСЂРµРІРµРґРёС‚Рµ СЂРµР·СЋРјРµ РІСЃС‚СЂРµС‡Рё РЅР° СЂСѓСЃСЃРєРёР№.',
]

const reportCalendarToday = new Date(2026, 5, 12)

const quickDateOptions = [
  { id: 'all', label: 'Р’ Р»СЋР±РѕРµ РІСЂРµРјСЏ' },
  { id: 'today', label: 'РЎРµРіРѕРґРЅСЏ', days: 1 },
  { id: 'last7', label: 'РџРѕСЃР»РµРґРЅРёРµ 7 РґРЅРµР№', days: 7 },
  { id: 'last30', label: 'РџРѕСЃР»РµРґРЅРёРµ 30 РґРЅРµР№', days: 30 },
  { id: 'last90', label: 'РџРѕСЃР»РµРґРЅРёРµ 90 РґРЅРµР№', days: 90 },
  { id: 'last6months', label: 'РџРѕСЃР»РµРґРЅРёРµ 6 РјРµСЃСЏС†РµРІ', months: 6 },
  { id: 'last12months', label: 'РџРѕСЃР»РµРґРЅРёРµ 12 РјРµСЃСЏС†РµРІ', months: 12 },
]

const typeFilterOptions = [
  { id: 'meetings', value: 'meeting', label: 'РћС‚С‡РµС‚С‹ Рѕ РІСЃС‚СЂРµС‡Р°С…', aliases: ['meeting', 'meetings', 'google meet'] },
  { id: 'readout', value: 'readout', label: 'РўРµРјС‹ Readout', aliases: ['readout'] },
  { id: 'daily', value: 'daily', label: 'Р•Р¶РµРґРЅРµРІРЅС‹Рµ РѕР±Р·РѕСЂС‹', aliases: ['daily'] },
]

const calendarMonthNames = [
  'СЏРЅРІР°СЂСЊ',
  'С„РµРІСЂР°Р»СЊ',
  'РјР°СЂС‚',
  'Р°РїСЂРµР»СЊ',
  'РјР°Р№',
  'РёСЋРЅСЊ',
  'РёСЋР»СЊ',
  'Р°РІРіСѓСЃС‚',
  'СЃРµРЅС‚СЏР±СЂСЊ',
  'РѕРєС‚СЏР±СЂСЊ',
  'РЅРѕСЏР±СЂСЊ',
  'РґРµРєР°Р±СЂСЊ',
]

const calendarShortMonthNames = ['СЏРЅРІ.', 'С„РµРІ.', 'РјР°СЂ.', 'Р°РїСЂ.', 'РјР°СЏ', 'РёСЋРЅ.', 'РёСЋР».', 'Р°РІРі.', 'СЃРµРЅ.', 'РѕРєС‚.', 'РЅРѕСЏ.', 'РґРµРє.']
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

const authSessionKey = 'alemlive-auth-session'
const authVerifierKey = 'alemlive-auth-verifier'
let currentAccessToken = ''

function setCurrentAccessToken(token) {
  currentAccessToken = token || ''
}

function getAuthHeaders() {
  return currentAccessToken ? { Authorization: `Bearer ${currentAccessToken}` } : {}
}

async function apiRequest(path, options = {}) {
  const isFormDataBody = typeof FormData !== 'undefined' && options.body instanceof FormData
  const response = await fetch(path, {
    ...options,
    headers: {
      ...(!isFormDataBody && options.body ? { 'Content-Type': 'application/json' } : {}),
      ...getAuthHeaders(),
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

function saveTextDownload(text, filename) {
  saveDownload(new Blob([text], { type: 'text/plain;charset=utf-8' }), filename)
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

  const [, reportId] = window.location.hash.match(/^#report\/(.+)$/) || []
  return reportId || ''
}

function getInitialView(initialReportId) {
  if (initialReportId) {
    return 'reportDetail'
  }

  if (typeof window !== 'undefined' && window.location.hash === '#meeting') {
    return 'meeting'
  }

  return 'reports'
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

  return 'РЈС‡Р°СЃС‚РЅРёРє'
}

function ParticipantsList({ participants }) {
  return (
    <section className="panel participants-panel">
      <div className="panel-heading">
        <span className="panel-icon">
          <Contact size={21} />
        </span>
        <div>
          <h2>РЈС‡Р°СЃС‚РЅРёРєРё</h2>
          <p>РљРѕРјР°РЅРґР° РІСЃС‚СЂРµС‡Рё</p>
        </div>
      </div>

      <div className="member-list">
        {participants.length === 0 ? (
          <p className="empty-members">РџРѕРєР° РЅРёРєС‚Рѕ РЅРµ РїСЂРёСЃРѕРµРґРёРЅРёР»СЃСЏ</p>
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
    return 'Р Р°Р·СЂРµС€РёС‚Рµ РґРѕСЃС‚СѓРї Рє РєР°РјРµСЂРµ Рё РјРёРєСЂРѕС„РѕРЅСѓ РІ Р±СЂР°СѓР·РµСЂРµ'
  }

  if (name === 'NotFoundError' || /not found|device not found/i.test(message)) {
    return 'РљР°РјРµСЂР° РёР»Рё РјРёРєСЂРѕС„РѕРЅ РЅРµ РЅР°Р№РґРµРЅС‹'
  }

  if (name === 'NotReadableError' || /busy|in use|could not start/i.test(message)) {
    return 'РљР°РјРµСЂР° РёР»Рё РјРёРєСЂРѕС„РѕРЅ Р·Р°РЅСЏС‚С‹ РґСЂСѓРіРёРј РїСЂРёР»РѕР¶РµРЅРёРµРј'
  }

  if (shouldWarnAboutMediaSecurity()) {
    return 'РћС‚РєСЂРѕР№С‚Рµ РІСЃС‚СЂРµС‡Сѓ С‡РµСЂРµР· HTTPS РёР»Рё localhost, РёРЅР°С‡Рµ Р±СЂР°СѓР·РµСЂ РјРѕР¶РµС‚ Р±Р»РѕРєРёСЂРѕРІР°С‚СЊ РєР°РјРµСЂСѓ Рё РјРёРєСЂРѕС„РѕРЅ'
  }

  return message || 'РќРµ СѓРґР°Р»РѕСЃСЊ РІРєР»СЋС‡РёС‚СЊ РєР°РјРµСЂСѓ РёР»Рё РјРёРєСЂРѕС„РѕРЅ'
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
        aria-label={isMicrophoneEnabled ? 'Р’С‹РєР»СЋС‡РёС‚СЊ РјРёРєСЂРѕС„РѕРЅ' : 'Р’РєР»СЋС‡РёС‚СЊ РјРёРєСЂРѕС„РѕРЅ'}
        aria-pressed={isMicrophoneEnabled}
      >
        {isMicrophoneEnabled ? <Mic size={18} /> : <MicOff size={18} />}
      </button>
      <button
        className={isCameraEnabled ? 'icon-button active' : 'icon-button'}
        type="button"
        onClick={() => toggleLiveKitDevice('camera')}
        disabled={pendingDevice === 'camera'}
        aria-label={isCameraEnabled ? 'Р’С‹РєР»СЋС‡РёС‚СЊ РєР°РјРµСЂСѓ' : 'Р’РєР»СЋС‡РёС‚СЊ РєР°РјРµСЂСѓ'}
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
      setError(sendError?.message || 'РќРµ СѓРґР°Р»РѕСЃСЊ РѕС‚РїСЂР°РІРёС‚СЊ СЃРѕРѕР±С‰РµРЅРёРµ')
    }
  }

  return (
    <section className="panel conference-chat-panel">
      <div className="panel-heading">
        <span className="panel-icon">
          <MessageSquareText size={21} />
        </span>
        <div>
          <h2>Р§Р°С‚ РІСЃС‚СЂРµС‡Рё</h2>
          <p>РЎРѕРѕР±С‰РµРЅРёСЏ LiveKit</p>
        </div>
        {onClose && (
          <button className="icon-button conference-chat-close" type="button" onClick={onClose} aria-label="РЎРєСЂС‹С‚СЊ С‡Р°С‚">
            <ChevronRight size={18} />
          </button>
        )}
      </div>

      <div className="conference-chat" role="log" aria-label="РЎРѕРѕР±С‰РµРЅРёСЏ РІСЃС‚СЂРµС‡Рё">
        <div className="conference-chat-messages">
          {chatMessages.length === 0 ? (
            <p className="conference-chat-empty">РџРѕРєР° СЃРѕРѕР±С‰РµРЅРёР№ РЅРµС‚</p>
          ) : (
            chatMessages.map((item) => {
              const author = item.from?.name || item.from?.identity || 'РЈС‡Р°СЃС‚РЅРёРє'
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
            placeholder="РќР°РїРёСЃР°С‚СЊ РІ С‡Р°С‚ РІСЃС‚СЂРµС‡Рё..."
            aria-label="РЎРѕРѕР±С‰РµРЅРёРµ РІ С‡Р°С‚ РІСЃС‚СЂРµС‡Рё"
          />
          <button className="ask-send" type="submit" disabled={isSending || !message.trim()} aria-label="РћС‚РїСЂР°РІРёС‚СЊ СЃРѕРѕР±С‰РµРЅРёРµ">
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
    userName: import.meta.env.VITE_LIVEKIT_NAME ?? 'РњР°РґРё РћСЂС‹СЃР±РµРє',
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
        const verifierPayload = JSON.parse(window.sessionStorage.getItem(authVerifierKey) || '{}')

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
        setWorkspaceNotice('Backend workspace РЅРµРґРѕСЃС‚СѓРїРµРЅ, РїРѕРєР°Р·Р°РЅС‹ Р»РѕРєР°Р»СЊРЅС‹Рµ РґР°РЅРЅС‹Рµ')
      }
    })

    return () => {
      isMounted = false
    }
  }, [authReady, isAuthenticated])

  useEffect(() => {
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
        if (nextReports.length && !nextReports.some((report) => report.id === selectedReportId)) {
          setSelectedReportId(nextReports[0].id)
        }
      } catch (error) {
        if (isMounted) {
          setReports(reportRows)
          setReportsError(error.message || 'РќРµ СѓРґР°Р»РѕСЃСЊ Р·Р°РіСЂСѓР·РёС‚СЊ РѕС‚С‡С‘С‚С‹ РёР· backend')
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
  }, [activeReportMode, reportSearchText, selectedReportId, selectedTypeFilterIds, timeFilterMode, timeFilterRange])

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

  useEffect(() => {
    if (typeof document === 'undefined') {
      return undefined
    }

    document.body.classList.toggle('meeting-maximized-active', isMeetingMaximized)
    return () => {
      document.body.classList.remove('meeting-maximized-active')
    }
  }, [isMeetingMaximized])

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

    setMeetingNotice('РЎРѕРµРґРёРЅРµРЅРёРµ СЃ РєРѕРјРЅР°С‚РѕР№ СЂР°Р·РѕСЂРІР°РЅРѕ')
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
        ...getAuthHeaders(),
      },
      body: JSON.stringify({ roomName, userName, isHost }),
    })

    const payload = await response.json().catch(() => ({}))

    if (!response.ok) {
      throw new Error(payload.error || 'РќРµ СѓРґР°Р»РѕСЃСЊ РїРѕР»СѓС‡РёС‚СЊ token РґР»СЏ РєРѕРјРЅР°С‚С‹')
    }

    if (!payload.serverUrl || !payload.token) {
      throw new Error('Backend РЅРµ РІРµСЂРЅСѓР» LiveKit URL РёР»Рё token')
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
      setJoinError('Р’РІРµРґРёС‚Рµ РёРјСЏ Рё РЅР°Р·РІР°РЅРёРµ РєРѕРјРЅР°С‚С‹')
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
    manualDisconnectRef.current = true
    await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/leave`, { method: 'POST' }).catch(() => null)
    await recordMeetingEvent('left')
    setMeeting(null)
    setIsMeetingMaximized(false)
    setMeetingNotice('')
  }

  async function copyRoom() {
    if (!navigator.clipboard) {
      return
    }

    await navigator.clipboard.writeText(getMeetingShareURL(meetingMeta.room))
    setWorkspaceNotice('РЎСЃС‹Р»РєР° РєРѕРјРЅР°С‚С‹ СЃРєРѕРїРёСЂРѕРІР°РЅР°')
  }

  async function copyRoomName() {
    if (!navigator.clipboard) {
      return
    }

    await navigator.clipboard.writeText(meetingMeta.room)
    setWorkspaceNotice('РќР°Р·РІР°РЅРёРµ РєРѕРјРЅР°С‚С‹ СЃРєРѕРїРёСЂРѕРІР°РЅРѕ')
  }

  async function copyRoomLink() {
    if (!navigator.clipboard) {
      return
    }

    await navigator.clipboard.writeText(getMeetingShareURL(meetingMeta.room))
    setWorkspaceNotice('РЎСЃС‹Р»РєР° РЅР° РєРѕРјРЅР°С‚Сѓ СЃРєРѕРїРёСЂРѕРІР°РЅР°')
  }

  async function showRoomSettings() {
    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/settings`).catch(() => null)
    if (payload) {
      setWorkspaceNotice(`Р—Р°РїРёСЃСЊ ${payload.recording ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}, Р°РІС‚РѕРѕС‚С‡С‘С‚ ${payload.autoReport ? 'РІРєР»СЋС‡РµРЅ' : 'РІС‹РєР»СЋС‡РµРЅ'}`)
    }
  }

  async function openRoomSettings() {
    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/settings`).catch(() => null)
    if (!payload) {
      return
    }

    setRoomSettings(payload)
    setIsRoomSettingsOpen((current) => !current)
    setWorkspaceNotice(`РќР°СЃС‚СЂРѕР№РєРё РєРѕРјРЅР°С‚С‹: Р·Р°РїРёСЃСЊ ${payload.recording ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}, Р°РІС‚РѕРѕС‚С‡С‘С‚ ${payload.autoReport ? 'РІРєР»СЋС‡С‘РЅ' : 'РІС‹РєР»СЋС‡РµРЅ'}`)
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
      setWorkspaceNotice(payload.items?.[0]?.body || 'РЈРІРµРґРѕРјР»РµРЅРёСЏ РѕР±РЅРѕРІР»РµРЅС‹')
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
      setWorkspaceNotice(`РЇР·С‹Рє: ${payload.items?.find((item) => item.id === payload.current)?.label || payload.current}`)
    }
  }

  function resetReportFilters() {
    setReportSearchText('')
    setActiveReportMode('reports')
    setTimeFilterMode('all')
    setTimeFilterRange({ from: null, to: null })
    setDraftTimeFilterRange({ from: null, to: null })
    setSelectedTypeFilterIds(typeFilterOptions.map((option) => option.id))
    setWorkspaceNotice('Р¤РёР»СЊС‚СЂС‹ СЃР±СЂРѕС€РµРЅС‹')
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
      setReportActionMessage(`Р—Р°РїРёСЃСЊ: ${payload.duration}, РјР°СЂРєРµСЂС‹ ${payload.markers?.join(', ') || 'РЅРµС‚'}`)
    }
  }

  function focusCopilotPanel() {
    setIsCopilotCollapsed(false)
    window.setTimeout(() => copilotInputRef.current?.focus(), 0)
    setReportActionMessage('Copilot РѕС‚РєСЂС‹С‚ Рё РіРѕС‚РѕРІ РѕС‚РІРµС‡Р°С‚СЊ РїРѕ РѕС‚С‡С‘С‚Сѓ')
  }

  function collapseCopilotPanel() {
    setIsCopilotCollapsed(true)
    setReportActionMessage('Copilot РјРѕР¶РЅРѕ СЃРІРµСЂРЅСѓС‚СЊ РЅР° СЃР»РµРґСѓСЋС‰РµРј С€Р°РіРµ UI')
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
        body: JSON.stringify({
          title: 'РќРѕРІР°СЏ РІСЃС‚СЂРµС‡Р°',
          source: 'Upload',
          owner: meetingMeta.name,
          folder: 'РћР±СЂР°Р±РѕС‚РєР°',
        }),
      })
      const nextReport = payload.report
      if (nextReport) {
        setReports((current) => [nextReport, ...current])
        setSelectedReportId(nextReport.id)
      }
      setReportActionMessage(payload.message || 'РћС‚С‡С‘С‚ РѕС‚РїСЂР°РІР»РµРЅ РЅР° РѕР±СЂР°Р±РѕС‚РєСѓ')
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
        body,
      })
      const nextReport = payload.report
      if (nextReport) {
        setReports((current) => [nextReport, ...current.filter((report) => report.id !== nextReport.id)])
        setSelectedReportId(nextReport.id)
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

    if (option.pending) {
      return `${option.label.replace(/\s*\(.+\)$/, '')} will be available after recording processing`
    }

    if (option.id === 'transcript') {
      const payload = await apiRequest(`/api/reports/${reportId}/transcript`)
      const detail = reportDetails[reportId]
      const report = detail?.report || reports.find((item) => item.id === reportId) || selectedReport
      const lines = payload.lines || detail?.transcriptLines || []
      const text = [
        report?.title || 'Meeting transcript',
        '',
        ...lines.map((line) => `${line.time || ''} ${line.speaker || ''}: ${line.text || ''}`.trim()),
      ].join('\n')
      saveTextDownload(text, filename)
      return 'Transcript download started'
    }

    const response = await fetch(`/api/reports/${reportId}/download`, {
      headers: getAuthHeaders(),
    })
    if (!response.ok) {
      const payload = await response.json().catch(() => ({}))
      throw new Error(payload.error || 'РќРµ СѓРґР°Р»РѕСЃСЊ СЃРєР°С‡Р°С‚СЊ РѕС‚С‡С‘С‚')
    }

    const blob = await response.blob()
    saveDownload(blob, filename)
    return 'РЎРєР°С‡РёРІР°РЅРёРµ РёС‚РѕРіР° РІСЃС‚СЂРµС‡Рё РЅР°С‡Р°Р»РѕСЃСЊ'
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
        setReportActionMessage('РЎСЃС‹Р»РєР° РЅР° РѕС‚С‡С‘С‚ СЃРєРѕРїРёСЂРѕРІР°РЅР°')
      } else if (actionId === 'rename') {
        const currentTitle = (reportDetails[reportId]?.report || reports.find((report) => report.id === reportId))?.title || ''
        const title = window.prompt('РќРѕРІРѕРµ РЅР°Р·РІР°РЅРёРµ РѕС‚С‡С‘С‚Р°', currentTitle)
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
        setReportActionMessage('РћС‚С‡С‘С‚ РїРµСЂРµРёРјРµРЅРѕРІР°РЅ')
      } else if (actionId === 'delete') {
        await apiRequest(`/api/reports/${reportId}`, { method: 'DELETE' })
        setReports((current) => current.filter((report) => report.id !== reportId))
        setReportActionMessage('РћС‚С‡С‘С‚ СѓРґР°Р»С‘РЅ')
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
        setReportActionMessage('РћС‚РїСЂР°РІРєР° РѕС‚С‡С‘С‚Р° РїРѕСЃС‚Р°РІР»РµРЅР° РІ РѕС‡РµСЂРµРґСЊ')
      } else if (actionId === 'copy') {
        const payload = await apiRequest(`/api/reports/${reportId}/copy`)
        if (navigator.clipboard) {
          await navigator.clipboard.writeText(payload.text || '')
        }
        setReportActionMessage('РўРµРєСЃС‚ РѕС‚С‡С‘С‚Р° СЃРєРѕРїРёСЂРѕРІР°РЅ')
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

    const searchQuery = kind === 'search' ? window.prompt('Р§С‚Рѕ РЅР°Р№С‚Рё РІ РѕС‚С‡С‘С‚Рµ?', 'backend') : ''
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
        setReportActionMessage(`Prompts: ${(payload.prompts || []).length}`)
      } else if (kind === 'history') {
        setReportActionMessage(`РСЃС‚РѕСЂРёСЏ С‡Р°С‚Р°: ${(payload.history || []).length}`)
      } else {
        setReportActionMessage(`РќР°Р№РґРµРЅРѕ: ${(payload.results || []).length}`)
      }
    } catch (error) {
      setReportActionMessage(error.message)
    }
  }

  async function copyReportNotes() {
    if (activeReportTab !== 'notes' || !selectedReportId) {
      setReportActionMessage('РљРѕРїРёСЂРѕРІР°РЅРёРµ Р·Р°РјРµС‚РѕРє РґРѕСЃС‚СѓРїРЅРѕ С‚РѕР»СЊРєРѕ РІРѕ РІРєР»Р°РґРєРµ Р—Р°РјРµС‚РєРё')
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
      setReportActionMessage('Р—Р°РјРµС‚РєРё СЃРєРѕРїРёСЂРѕРІР°РЅС‹')
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
      setReportActionMessage('Р—Р°РјРµС‚РєРё РѕС‚РїСЂР°РІР»РµРЅС‹ РЅР° СЂРµРґР°РєС‚РёСЂРѕРІР°РЅРёРµ')
    } catch (error) {
      setReportActionMessage(error.message || 'Backend endpoint РґР»СЏ СЂРµРґР°РєС‚РёСЂРѕРІР°РЅРёСЏ Р·Р°РјРµС‚РѕРє РїРѕРєР° РЅРµРґРѕСЃС‚СѓРїРµРЅ')
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
      setCopilotMessages((current) => [...current, { role: 'assistant', text: payload.answer || 'РћС‚РІРµС‚ РїСѓСЃС‚РѕР№' }])
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
          <span>Р’С‹Р±СЂР°С‚СЊ РІСЃРµ</span>
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
            <strong>{calendarMonthNames[calendarMonth.getMonth()]} {calendarMonth.getFullYear()} Рі.</strong>
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
            <p>{isConnected ? 'LiveKit conference Р·Р°РїСѓС‰РµРЅР°' : 'РћР¶РёРґР°РµС‚ РїРѕРґРєР»СЋС‡РµРЅРёСЏ'}</p>
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
              aria-label={isMeetingMaximized ? 'РЎРІРµСЂРЅСѓС‚СЊ РІРёРґРµРѕРєРѕРЅС„РµСЂРµРЅС†РёСЋ' : 'Р Р°Р·РІРµСЂРЅСѓС‚СЊ РІРёРґРµРѕРєРѕРЅС„РµСЂРµРЅС†РёСЋ'}
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
              aria-label={isConferenceChatOpen ? 'РЎРєСЂС‹С‚СЊ С‡Р°С‚ РІСЃС‚СЂРµС‡Рё' : 'РџРѕРєР°Р·Р°С‚СЊ С‡Р°С‚ РІСЃС‚СЂРµС‡Рё'}
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
              Р—Р°РІРµСЂС€РёС‚СЊ
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
            <h1 id="meeting-title">РљРѕРјРЅР°С‚Р° РґР»СЏ СЃРѕР·РІРѕРЅР°</h1>
            <p>РЎРѕР·РґР°Р№С‚Рµ РЅРѕРІСѓСЋ РєРѕРјРЅР°С‚Сѓ РѕРґРЅРёРј РЅР°Р¶Р°С‚РёРµРј РёР»Рё РїРѕРґРєР»СЋС‡РёС‚РµСЃСЊ РїРѕ РЅР°Р·РІР°РЅРёСЋ СѓР¶Рµ СЃСѓС‰РµСЃС‚РІСѓСЋС‰РµР№ РєРѕРјРЅР°С‚С‹.</p>
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
              РЎРѕР·РґР°С‚СЊ РєРѕРјРЅР°С‚Сѓ
            </button>
            <button className="soft-action" type="button" onClick={() => selectEntryMode('join')}>
              <Link size={18} />
              РџСЂРёСЃРѕРµРґРёРЅРёС‚СЊСЃСЏ
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
                  <p>{isConnected ? 'РљРѕРјРЅР°С‚Р° Р°РєС‚РёРІРЅР°' : 'URL Рё token Р±СѓРґСѓС‚ РїРѕР»СѓС‡РµРЅС‹ Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєРё'}</p>
                </div>
              </div>

              <form className="join-form" onSubmit={joinMeeting}>
                <div className="entry-switch" aria-label="Р’С‹Р±РµСЂРёС‚Рµ РґРµР№СЃС‚РІРёРµ">
                  <button
                    className={entryMode === 'create' ? 'entry-option active' : 'entry-option'}
                    type="button"
                    onClick={() => selectEntryMode('create')}
                  >
                    <Sparkles size={18} />
                    РЎРѕР·РґР°С‚СЊ РєРѕРјРЅР°С‚Сѓ
                  </button>
                  <button
                    className={entryMode === 'join' ? 'entry-option active' : 'entry-option'}
                    type="button"
                    onClick={() => selectEntryMode('join')}
                  >
                    <Link size={18} />
                    РџСЂРёСЃРѕРµРґРёРЅРёС‚СЊСЃСЏ
                  </button>
                </div>

                <label>
                  <span>Р’Р°С€Рµ РёРјСЏ</span>
                  <input name="userName" value={form.userName} onChange={updateField} autoComplete="name" />
                </label>

                <label>
                  <span>РќР°Р·РІР°РЅРёРµ РєРѕРјРЅР°С‚С‹</span>
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
                    Р”Р»СЏ РєР°РјРµСЂС‹ Рё РјРёРєСЂРѕС„РѕРЅР° РѕС‚РєСЂРѕР№С‚Рµ РІСЃС‚СЂРµС‡Сѓ С‡РµСЂРµР· HTTPS РёР»Рё localhost.
                  </p>
                )}
                {meetingNotice && <p className="form-error">{meetingNotice}</p>}

                <button className="join-button" type="submit" disabled={!canStart || isStarting}>
                  {isStarting ? <Loader2 className="spin-icon" size={18} /> : <Play size={18} fill="currentColor" />}
                  {isConnected
                    ? 'РџРµСЂРµРїРѕРґРєР»СЋС‡РёС‚СЊСЃСЏ'
                    : entryMode === 'create'
                      ? 'РЎРѕР·РґР°С‚СЊ Рё РІРѕР№С‚Рё'
                      : 'Р’РѕР№С‚Рё РїРѕ РЅР°Р·РІР°РЅРёСЋ'}
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
                    <h2>РџРµСЂРµРґ РІС…РѕРґРѕРј</h2>
                    <p>Р’С‹Р±РµСЂРёС‚Рµ, С‡С‚Рѕ РІРєР»СЋС‡РёС‚СЊ СЃСЂР°Р·Сѓ РІ РєРѕРјРЅР°С‚Рµ</p>
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
                    <span>РњРёРєСЂРѕС„РѕРЅ</span>
                    <strong>{devices.mic ? 'РІРєР»СЋС‡РµРЅ' : 'РІС‹РєР»СЋС‡РµРЅ'}</strong>
                  </button>
                  <button
                    className={devices.camera ? 'device-toggle active' : 'device-toggle'}
                    type="button"
                    onClick={() => toggleDevice('camera')}
                    aria-pressed={devices.camera}
                  >
                    {devices.camera ? <Video size={19} /> : <CameraOff size={19} />}
                    <span>РљР°РјРµСЂР°</span>
                    <strong>{devices.camera ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}</strong>
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
                    <span>Р—Р°РїРёСЃСЊ: {roomSettings.recording ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}</span>
                    <span>РўСЂР°РЅСЃРєСЂРёРїС†РёСЏ: {roomSettings.transcription ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}</span>
                    <span>Р“РѕСЃС‚Рё: {roomSettings.allowGuests ? 'СЂР°Р·СЂРµС€РµРЅС‹' : 'Р·Р°РїСЂРµС‰РµРЅС‹'}</span>
                    <span>РђРІС‚РѕРѕС‚С‡С‘С‚: {roomSettings.autoReport ? 'РІРєР»СЋС‡С‘РЅ' : 'РІС‹РєР»СЋС‡РµРЅ'}</span>
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
                    <span>Р—Р°РїРёСЃСЊ: {roomSettings.recording ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}</span>
                    <span>РўСЂР°РЅСЃРєСЂРёРїС†РёСЏ: {roomSettings.transcription ? 'РІРєР»СЋС‡РµРЅР°' : 'РІС‹РєР»СЋС‡РµРЅР°'}</span>
                    <span>Р“РѕСЃС‚Рё: {roomSettings.allowGuests ? 'СЂР°Р·СЂРµС€РµРЅС‹' : 'Р·Р°РїСЂРµС‰РµРЅС‹'}</span>
                    <span>РђРІС‚РѕРѕС‚С‡С‘С‚: {roomSettings.autoReport ? 'РІРєР»СЋС‡С‘РЅ' : 'РІС‹РєР»СЋС‡РµРЅ'}</span>
                  </div>
                )}

                <div className="livekit-stage">
                  <div className="empty-meeting">
                    <div className="empty-orbit">
                      <Bot size={34} />
                    </div>
                    <h3>{entryMode === 'create' ? 'Р“РѕС‚РѕРІРѕ Рє СЃРѕР·РґР°РЅРёСЋ РєРѕРјРЅР°С‚С‹' : 'Р“РѕС‚РѕРІРѕ Рє РїРѕРґРєР»СЋС‡РµРЅРёСЋ'}</h3>
                    <p>
                      {entryMode === 'create'
                        ? 'РќР°Р¶РјРёС‚Рµ СЃРѕР·РґР°С‚СЊ РєРѕРјРЅР°С‚Сѓ, Рё РїСЂРёР»РѕР¶РµРЅРёРµ СЃР°РјРѕ РїРѕР»СѓС‡РёС‚ LiveKit token С‡РµСЂРµР· backend.'
                        : 'Р’РІРµРґРёС‚Рµ РЅР°Р·РІР°РЅРёРµ РєРѕРјРЅР°С‚С‹ Рё РІРѕР№РґРёС‚Рµ. URL Рё token РІРІРѕРґРёС‚СЊ РІСЂСѓС‡РЅСѓСЋ Р±РѕР»СЊС€Рµ РЅРµ РЅСѓР¶РЅРѕ.'}
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
          <h1>РћС‚С‡С‘С‚С‹</h1>
        </div>

        <div className="ask-read-bar">
          <button className="ask-locale" type="button" onClick={refreshLocales}>
            <Grid2X2 size={18} />
            <span>{locales.current?.toUpperCase?.() || 'RU'}</span>
            <ChevronDown size={16} />
          </button>
          <span>РЎРїСЂРѕСЃРёС‚Рµ Alem Рѕ С‡С‘Рј СѓРіРѕРґРЅРѕ...</span>
          <button className="ask-send" type="button" onClick={openAskAI} aria-label="Send question">
            <Send size={19} />
          </button>
        </div>

        <div className="reports-subnav">
          <div className="report-mode-tabs">
            <button className={activeReportMode === 'reports' ? 'report-mode active' : 'report-mode'} type="button" onClick={() => setActiveReportMode('reports')}>РћС‚С‡С‘С‚С‹</button>
            <button className={activeReportMode === 'incomplete' ? 'report-mode active' : 'report-mode'} type="button" onClick={() => setActiveReportMode('incomplete')}>РќРµРїРѕР»РЅС‹Р№</button>
          </div>
          <div className="last-updated">
            <RefreshCw size={16} />
            РџРѕСЃР»РµРґРЅРµРµ РѕР±РЅРѕРІР»РµРЅРёРµ РІ 15:37
          </div>
          <button className="primary-action upload-action" type="button" onClick={uploadReport}>
            <Download size={18} />
            Р—Р°РіСЂСѓР·РёС‚СЊ
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
              placeholder="Р¤РёР»СЊС‚СЂ РїРѕ РЅР°Р·РІР°РЅРёСЋ РѕС‚С‡С‘С‚Р°"
            />
          </label>
          <button className="filter-button" type="button" onClick={resetReportFilters}>
            <FileText size={17} />
            Р’СЃРµ РѕС‚С‡С‘С‚С‹
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
              РўРёРї
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
            <span>РСЃС‚РѕС‡РЅРёРє</span>
            <span>РћС‚С‡С‘С‚</span>
            <span>
              Р”Р°С‚Р° Рё РІСЂРµРјСЏ
              <ArrowDown size={17} />
            </span>
            <span>РџР°РїРєРё</span>
            <span>Р’Р»Р°РґРµР»РµС†</span>
          </div>

          <div className="reports-week">{reportsLoading ? 'Р—РђР“Р РЈР—РљРђ РћРўР§РЃРўРћР’...' : visibleReports[0]?.week || 'РћРўР§РЃРўР«'}</div>

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
                    aria-label={`Р”РµР№СЃС‚РІРёСЏ РґР»СЏ РѕС‚С‡С‘С‚Р° ${report.title}`}
                    aria-expanded={openReportActionsId === report.id}
                  >
                    <MoreHorizontal size={22} />
                  </button>
                  {openReportActionsId === report.id && (
                    <span className="report-actions-menu" onClick={keepReportActionsOpen}>
                      {(reportActions[report.id] || [
                        { id: 'share', label: 'РџРѕРґРµР»РёС‚СЊСЃСЏ', enabled: true },
                        { id: 'download', label: 'РЎРєР°С‡Р°С‚СЊ', enabled: true },
                        { id: 'rename', label: 'РџРµСЂРµРёРјРµРЅРѕРІР°С‚СЊ РѕС‚С‡РµС‚', enabled: true },
                        { id: 'delete', label: 'РЈРґР°Р»РёС‚СЊ РѕС‚С‡РµС‚', enabled: true, danger: true },
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
              <span>РћС†РµРЅРєР° Alem</span>
              <strong>{selectedReport.score}</strong>
              <small>РҐРћР РћРЁРћ</small>
            </div>
            <div>
              <span>Р’РѕРІР»РµС‡С‘РЅРЅРѕСЃС‚СЊ</span>
              <strong>93</strong>
              <small>РҐРћР РћРЁРћ</small>
            </div>
            <div>
              <span>РќР°СЃС‚СЂРѕРµРЅРёРµ</span>
              <strong>85</strong>
              <small>РҐРћР РћРЁРћ</small>
            </div>
          </div>

          <section className="report-main-panel">
            <div className="section-kicker">
              <Sparkles size={18} />
              РЎРІРѕРґРєР°
            </div>
            <span className="edited-pill">РћС‚СЂРµРґР°РєС‚РёСЂРѕРІР°РЅРѕ</span>
            <h3>{detailSummary[0]?.title || 'РљРѕРјР°РЅРґР° СЃРѕРіР»Р°СЃРѕРІР°Р»Р° РЅРѕРІС‹Р№ СЃС†РµРЅР°СЂРёР№ РІС…РѕРґР° Рё СЃС‚СЂСѓРєС‚СѓСЂСѓ AI РѕС‚С‡С‘С‚Р°'}</h3>
            <p>
              {detailSummary[0]?.text || 'Р’СЃС‚СЂРµС‡Р° Р±С‹Р»Р° РїРѕСЃРІСЏС‰РµРЅР° РЅР°СЃС‚СЂРѕР№РєРµ AlemLive Рё Р°РЅР°Р»РёС‚РёС‡РµСЃРєРѕРіРѕ РѕС‚С‡С‘С‚Р° РїРѕСЃР»Рµ СЃРѕР·РІРѕРЅР°. РЈС‡Р°СЃС‚РЅРёРєРё РґРѕРіРѕРІРѕСЂРёР»РёСЃСЊ, С‡С‚Рѕ РїРѕР»СЊР·РѕРІР°С‚РµР»СЊ РґРѕР»Р¶РµРЅ СЃРѕР·РґР°РІР°С‚СЊ РєРѕРјРЅР°С‚Сѓ Рё РїСЂРёСЃРѕРµРґРёРЅСЏС‚СЊСЃСЏ РїРѕ РЅР°Р·РІР°РЅРёСЋ, Р° URL Рё token РґРѕР»Р¶РЅС‹ РїРѕРґС‚СЏРіРёРІР°С‚СЊСЃСЏ Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєРё С‡РµСЂРµР· backend. РџРѕСЃР»Рµ РІСЃС‚СЂРµС‡Рё Р°РіРµРЅС‚ РїРѕРєР°Р·С‹РІР°РµС‚ СЂРµР·СЋРјРµ, Р·Р°РґР°С‡Рё, С‚СЂР°РЅСЃРєСЂРёРїС‚, РјРµС‚СЂРёРєРё Рё РіР»Р°РІС‹.'}
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
              <span>РџРѕРёСЃРє РїРѕ С‚СЂР°РЅСЃРєСЂРёРїС‚Сѓ: token, РєРѕРјРЅР°С‚Р°, РѕС‚С‡С‘С‚</span>
            </div>
            <span className="report-badge muted">{detailTranscriptLines.length} РјРѕРјРµРЅС‚РѕРІ</span>
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
              <p>РџРѕР·РёС‚РёРІРЅР°СЏ РґРёРЅР°РјРёРєР°</p>
            </div>
            <div className="metric-card">
              <Zap size={20} />
              <span>Engagement</span>
              <strong>74%</strong>
              <p>Р’С‹СЃРѕРєРѕРµ СѓС‡Р°СЃС‚РёРµ</p>
            </div>
            <div className="metric-card">
              <Clock3 size={20} />
              <span>Interruptions</span>
              <strong>3</strong>
              <p>РќРёР·РєРёР№ СѓСЂРѕРІРµРЅСЊ РїРµСЂРµР±РёРІР°РЅРёР№</p>
            </div>
          </div>
          <div className="speaker-table">
            {detailSpeakerStats.map((speaker) => (
              <article className="speaker-row" key={speaker.name}>
                <div>
                  <strong>{speaker.name}</strong>
                  <span>{speaker.sentiment} В· {speaker.pace}</span>
                </div>
                <div className="talk-bar" aria-label={`${speaker.name} РіРѕРІРѕСЂРёР» ${speaker.talk || speaker.talkTime}% РІСЂРµРјРµРЅРё`}>
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
                  {selectedReport.participantNames || 'Alison Barker, РњР°РґРё, РђР№РґР°РЅР°, +1 Р±РѕР»СЊС€Рµ'}
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
                РЎРєР°С‡Р°С‚СЊ
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
              РћС‚РїСЂР°РІРёС‚СЊ РІ...
            </button>
            <button className="soft-action" type="button" onClick={() => handleReportAction(selectedReport.id, 'share')}>
              <Share2 size={18} />
              РџРѕРґРµР»РёС‚СЊСЃСЏ
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
              <div className="detail-tab-list" role="tablist" aria-label="Р Р°Р·РґРµР»С‹ РѕС‚С‡С‘С‚Р°">
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
                <button className="detail-tool-button" type="button" onClick={() => runReportLookup('search')} aria-label="РџРѕРёСЃРє РїРѕ РѕС‚С‡С‘С‚Сѓ">
                  <Search size={21} />
                </button>
                <button
                  className="detail-tool-button"
                  type="button"
                  onClick={copyReportNotes}
                  disabled={activeReportTab !== 'notes'}
                  aria-label="РљРѕРїРёСЂРѕРІР°С‚СЊ Р·Р°РјРµС‚РєРё"
                >
                  <Copy size={21} />
                </button>
                <div className="detail-more-wrap">
                  <button
                    className={isDetailActionsOpen ? 'detail-tool-button active' : 'detail-tool-button'}
                    type="button"
                    onClick={() => setIsDetailActionsOpen((current) => !current)}
                    aria-label="Р”РѕРїРѕР»РЅРёС‚РµР»СЊРЅС‹Рµ РґРµР№СЃС‚РІРёСЏ"
                    aria-expanded={isDetailActionsOpen}
                  >
                    <MoreHorizontal size={22} />
                  </button>
                  {isDetailActionsOpen && (
                    <div className="detail-more-menu">
                      <button className="detail-more-item" type="button" onClick={editReportNotes}>
                        <Edit3 size={18} />
                        Р РµРґР°РєС‚РёСЂРѕРІР°С‚СЊ Р·Р°РјРµС‚РєРё
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
                placeholder="РЎРїСЂРѕСЃРёС‚Рµ Alem Рѕ С‡С‘Рј СѓРіРѕРґРЅРѕ..."
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
          <p>{authReady ? 'Р’РѕР№РґРёС‚Рµ С‡РµСЂРµР· Keycloak, С‡С‚РѕР±С‹ РїСЂРѕРґРѕР»Р¶РёС‚СЊ.' : 'РџСЂРѕРІРµСЂСЏРµРј Р°РІС‚РѕСЂРёР·Р°С†РёСЋ...'}</p>
          {authError && <p className="auth-error">{authError}</p>}
          {authReady && (
            <button className="primary-action" type="button" onClick={loginWithKeycloak}>
              <Lock size={18} />
              Login with Keycloak
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
