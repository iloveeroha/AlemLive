import { useEffect, useMemo, useRef, useState } from 'react'
import {
  ArrowDown,
  ArrowDownUp,
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
  ChevronUp,
  Clock3,
  Contact,
  Copy,
  Download,
  Edit3,
  ExternalLink,
  FileText,
  Filter,
  Folder,
  Flag,
  Globe,
  Grid2X2,
  GripVertical,
  HelpCircle,
  Highlighter,
  Info,
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
  Square,
  Trash2,
  TrendingUp,
  Users,
  Video,
  X,
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

const fallbackReportActions = [
  { id: 'share', label: 'Поделиться', enabled: true },
  { id: 'download', label: 'Скачать', enabled: true },
  { id: 'rename', label: 'Переименовать отчет', enabled: true },
  { id: 'delete', label: 'Удалить отчет', enabled: true, danger: true },
]

const speakerAvatarPalette = ['#16a34a', '#7c3aed', '#ea580c', '#db2777', '#2563eb', '#0d9488', '#ca8a04', '#dc2626']

function speakerInitials(name) {
  const trimmed = (name || '').trim()
  if (!trimmed) {
    return '?'
  }
  const parts = trimmed.split(/\s+/).filter(Boolean)
  const initials = parts.slice(0, 2).map((part) => part[0]?.toUpperCase() || '')
  return initials.join('') || trimmed[0].toUpperCase()
}

function speakerAvatarColor(name) {
  const key = (name || '').trim().toLowerCase()
  let hash = 0
  for (let i = 0; i < key.length; i += 1) {
    hash = (hash * 31 + key.charCodeAt(i)) % speakerAvatarPalette.length
  }
  return speakerAvatarPalette[((hash % speakerAvatarPalette.length) + speakerAvatarPalette.length) % speakerAvatarPalette.length]
}

function timeToSeconds(value) {
  const parts = (value || '0:00').split(':').map((part) => parseInt(part, 10) || 0)
  if (parts.length === 3) {
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  }
  if (parts.length === 2) {
    return parts[0] * 60 + parts[1]
  }
  return parts[0] || 0
}

function highlightKey(item) {
  return `${item.time}-${item.title}`
}

function chapterKey(chapter) {
  return `${chapter.start || chapter.time}-${chapter.title}`
}

function formatDurationLabel(seconds) {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return null
  }
  const minutes = Math.round(seconds / 60)
  if (minutes < 1) {
    return `${Math.round(seconds)} с`
  }
  return `${minutes} мин`
}

function chapterDurationSeconds(chapter, nextChapterStartSeconds) {
  const startSeconds = timeToSeconds(chapter.start || chapter.time)
  if (chapter.end) {
    return timeToSeconds(chapter.end) - startSeconds
  }
  if (Number.isFinite(nextChapterStartSeconds)) {
    return nextChapterStartSeconds - startSeconds
  }
  return null
}

function captureVideoFrame(video, seconds) {
  return new Promise((resolve, reject) => {
    if (!video || Number.isNaN(seconds)) {
      reject(new Error('Invalid video or timestamp'))
      return
    }

    function cleanup() {
      video.removeEventListener('seeked', onSeeked)
      video.removeEventListener('error', onError)
    }
    function onSeeked() {
      cleanup()
      try {
        const canvas = document.createElement('canvas')
        canvas.width = video.videoWidth || 320
        canvas.height = video.videoHeight || 180
        const ctx = canvas.getContext('2d')
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
        resolve(canvas.toDataURL('image/jpeg', 0.7))
      } catch (error) {
        reject(error)
      }
    }
    function onError() {
      cleanup()
      reject(new Error('Video failed to seek'))
    }

    video.addEventListener('seeked', onSeeked)
    video.addEventListener('error', onError)
    video.currentTime = Math.max(0, seconds)
  })
}

function groupTranscriptByChapters(lines, chapterList) {
  if (!Array.isArray(chapterList) || chapterList.length === 0) {
    return [{ chapter: null, lines }]
  }

  const sortedChapters = chapterList
    .map((chapter) => ({ ...chapter, startSeconds: timeToSeconds(chapter.start || chapter.time) }))
    .sort((a, b) => a.startSeconds - b.startSeconds)

  const groups = sortedChapters.map((chapter) => ({ chapter, lines: [] }))
  const ungrouped = []

  lines.forEach((line) => {
    const lineSeconds = timeToSeconds(line.time)
    let target = null
    for (let i = sortedChapters.length - 1; i >= 0; i -= 1) {
      if (lineSeconds >= sortedChapters[i].startSeconds) {
        target = groups[i]
        break
      }
    }
    if (target) {
      target.lines.push(line)
    } else {
      ungrouped.push(line)
    }
  })

  const result = ungrouped.length > 0 ? [{ chapter: null, lines: ungrouped }] : []
  return [...result, ...groups.filter((group) => group.lines.length > 0)]
}

function moodDescription(value) {
  const score = Number(value) || 0
  if (score >= 80) return 'Позитивная динамика'
  if (score >= 50) return 'Нейтральная динамика'
  return 'Настроение требует внимания'
}

function engagementDescription(value) {
  const score = Number(value) || 0
  if (score >= 80) return 'Высокое участие'
  if (score >= 50) return 'Среднее участие'
  return 'Низкое участие'
}

function interruptionsDescription(count) {
  const value = Number(count) || 0
  if (value === 0) return 'Перебиваний не обнаружено'
  if (value <= 3) return 'Низкий уровень перебиваний'
  if (value <= 7) return 'Средний уровень перебиваний'
  return 'Высокий уровень перебиваний'
}

function highlightTypeMeta(type) {
  const normalized = (type || '').toLowerCase()
  if (normalized === 'question') {
    return { label: 'Ключевой вопрос', className: 'highlight-tag-question', Icon: HelpCircle }
  }
  if (normalized === 'action') {
    return { label: 'Действие', className: 'highlight-tag-action', Icon: Flag }
  }
  return { label: 'Тема', className: 'highlight-tag-topic', Icon: MessageSquareText }
}

function scoreLabel(value) {
  const score = Number(value) || 0
  if (score >= 80) {
    return 'ХОРОШО'
  }
  if (score >= 50) {
    return 'СРЕДНЕ'
  }
  return 'НИЗКО'
}

function transcriptLineKey(line) {
  return line.id || `${line.time}-${line.speaker}`
}

function findTranscriptMatches(chapterGroups, query) {
  const trimmed = query.trim().toLowerCase()
  if (!trimmed) {
    return []
  }

  const matches = []
  chapterGroups.forEach((group) => {
    const chapterKey = group.chapter?.title || 'ungrouped'
    group.lines.forEach((line) => {
      const lower = (line.text || '').toLowerCase()
      let from = 0
      let idx = lower.indexOf(trimmed, from)
      while (idx !== -1) {
        matches.push({ lineKey: transcriptLineKey(line), chapterKey, charIndex: idx })
        from = idx + trimmed.length
        idx = lower.indexOf(trimmed, from)
      }
    })
  })
  return matches
}

function splitTextForHighlight(text, query, activeCharIndex) {
  const value = text || ''
  const trimmed = query.trim()
  if (!trimmed) {
    return [{ text: value, match: false }]
  }

  const lower = value.toLowerCase()
  const needle = trimmed.toLowerCase()
  const segments = []
  let cursor = 0
  let idx = lower.indexOf(needle, cursor)
  if (idx === -1) {
    return [{ text: value, match: false }]
  }

  while (idx !== -1) {
    if (idx > cursor) {
      segments.push({ text: value.slice(cursor, idx), match: false })
    }
    segments.push({ text: value.slice(idx, idx + needle.length), match: true, active: idx === activeCharIndex })
    cursor = idx + needle.length
    idx = lower.indexOf(needle, cursor)
  }
  if (cursor < value.length) {
    segments.push({ text: value.slice(cursor), match: false })
  }
  return segments
}

const trendLineColors = { score: '#5c4df4', engagement: '#ef4444', mood: '#d946ef' }

function niceAxisBounds(values) {
  const min = Math.min(...values)
  const max = Math.max(...values)
  const lower = Math.max(0, Math.floor(min / 10) * 10 - 10)
  const upper = Math.min(100, Math.ceil(max / 10) * 10 + 10)
  return lower === upper ? [Math.max(0, lower - 10), Math.min(100, upper + 10)] : [lower, upper]
}

function smoothPath(coords) {
  if (coords.length < 2) {
    return ''
  }
  let path = `M ${coords[0][0]} ${coords[0][1]}`
  for (let i = 0; i < coords.length - 1; i += 1) {
    const [x0, y0] = coords[i]
    const [x1, y1] = coords[i + 1]
    const midX = (x0 + x1) / 2
    path += ` Q ${x0} ${y0} ${midX} ${(y0 + y1) / 2} Q ${x1} ${y1} ${x1} ${y1}`
  }
  return path
}

function TrendChart({ points }) {
  if (!Array.isArray(points) || points.length < 2) {
    return null
  }

  const width = 720
  const height = 220
  const allValues = points.flatMap((point) => [point.mood, point.engagement, point.score])
  const [minValue, maxValue] = niceAxisBounds(allValues)
  const stepX = width / (points.length - 1)
  const toY = (value) => height - ((value - minValue) / (maxValue - minValue)) * height
  const coordsFor = (key) => points.map((point, index) => [index * stepX, toY(point[key])])
  const buildLine = (key) => smoothPath(coordsFor(key))
  const buildArea = (key) => {
    const coords = coordsFor(key)
    return `${smoothPath(coords)} L ${coords[coords.length - 1][0]} ${height} L ${coords[0][0]} ${height} Z`
  }

  const gridLines = [minValue, (minValue + maxValue) / 2, maxValue]

  return (
    <svg className="trend-chart" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none">
      <defs>
        <linearGradient id="trend-fill-mood" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={trendLineColors.mood} stopOpacity="0.28" />
          <stop offset="100%" stopColor={trendLineColors.mood} stopOpacity="0" />
        </linearGradient>
        <linearGradient id="trend-fill-engagement" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={trendLineColors.engagement} stopOpacity="0.28" />
          <stop offset="100%" stopColor={trendLineColors.engagement} stopOpacity="0" />
        </linearGradient>
        <linearGradient id="trend-fill-score" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={trendLineColors.score} stopOpacity="0.28" />
          <stop offset="100%" stopColor={trendLineColors.score} stopOpacity="0" />
        </linearGradient>
      </defs>

      {gridLines.map((value) => (
        <line key={value} className="trend-gridline" x1="0" y1={toY(value)} x2={width} y2={toY(value)} />
      ))}

      <path d={buildArea('engagement')} fill="url(#trend-fill-engagement)" stroke="none" />
      <path d={buildArea('mood')} fill="url(#trend-fill-mood)" stroke="none" />
      <path d={buildArea('score')} fill="url(#trend-fill-score)" stroke="none" />

      <path d={buildLine('mood')} fill="none" stroke={trendLineColors.mood} strokeWidth="2" />
      <path d={buildLine('engagement')} fill="none" stroke={trendLineColors.engagement} strokeWidth="2" />
      <path d={buildLine('score')} fill="none" stroke={trendLineColors.score} strokeWidth="2" />

      {gridLines.map((value) => (
        <text key={`label-${value}`} className="trend-gridline-label" x={width} y={toY(value) - 4}>{Math.round(value)}</text>
      ))}
    </svg>
  )
}

function PositionSlider({ label, description, value, hasValue, average, min, max, unit }) {
  const range = max - min
  const clampPercent = (raw) => Math.max(0, Math.min(100, ((raw - min) / range) * 100))
  const averagePercent = clampPercent(average)
  const valuePercent = hasValue ? clampPercent(value) : null

  return (
    <article className="position-row">
      <div className="position-info">
        <strong>{label}</strong>
        <p>{description}</p>
      </div>
      <div className="position-track-wrap">
        {hasValue && (
          <span className="position-value" style={{ left: `${valuePercent}%` }}>{value}{unit}</span>
        )}
        <div className="position-track">
          <span className="position-track-fill" style={{ width: `${averagePercent}%` }} />
          {hasValue && <span className="position-track-dot" style={{ left: `${valuePercent}%` }} />}
        </div>
        <div className="position-track-labels">
          <span>{min}{unit}</span>
          <span>{max}{unit}</span>
        </div>
      </div>
    </article>
  )
}

function averageOf(values) {
  if (!values.length) {
    return 0
  }
  return Math.round(values.reduce((sum, value) => sum + value, 0) / values.length)
}

function normalizeReportActions(actions) {
  const incomingActions = Array.isArray(actions) ? actions : []
  const byID = new Map(fallbackReportActions.map((action) => [action.id, action]))

  incomingActions.forEach((action) => {
    if (action?.id) {
      byID.set(action.id, { ...byID.get(action.id), ...action })
    }
  })

  return Array.from(byID.values())
}

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
  {
    time: '00:00', title: 'Старт и цель встречи', duration: '4 мин',
    text: 'Команда обозначила цель звонка и кратко прошлась по плану.',
    points: ['Цель и план встречи', 'Кто участвует и зачем'],
  },
  {
    time: '04:01', title: 'LiveKit вход и комнаты', duration: '9 мин',
    text: 'Обсудили, как пользователь создаёт и подключается к комнате LiveKit.',
    points: ['Создание комнаты по названию', 'Автоматическая выдача токена backend'],
  },
  {
    time: '13:10', title: 'Структура AI отчёта', duration: '12 мин',
    text: 'Разобрали вкладки отчёта: заметки, транскрипт, главы и основные моменты.',
    points: ['Сводка и action items', 'Главы, транскрипт и основные моменты'],
  },
  {
    time: '25:30', title: 'Action items и финальные решения', duration: '7 мин',
    text: 'Зафиксировали итоговые задачи и договорённости команды.',
    points: ['Распределение задач по владельцам', 'Сроки и приоритеты'],
  },
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
  const recording = report?.recordingStatus || ''
  const transcription = report?.transcriptionStatus || ''
  const analysis = report?.analysisStatus || ''
  if (recording === 'running') {
    return 'Запись идет'
  }
  if (recording === 'pending') {
    return 'Запись ожидает'
  }
  if (recording === 'failed') {
    return 'Ошибка записи'
  }
  if (transcription === 'pending' || transcription === 'running') {
    return 'Транскрипция'
  }
  if (transcription === 'failed') {
    return 'Ошибка STT'
  }
  if (analysis === 'pending' || analysis === 'running') {
    return 'AI-анализ'
  }
  if (analysis === 'failed') {
    return 'AI fallback'
  }
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

function roomRecordingLabel(state) {
  switch (state) {
    case 'recording':
      return 'Запись идет'
    case 'processing':
      return 'Останавливается'
    case 'ready':
      return 'Запись готова'
    case 'error':
      return 'Ошибка записи'
    default:
      return 'Запись'
  }
}

const authSessionKey = 'alemlive-auth-session-v2'
const authVerifierKey = 'alemlive-auth-verifier'
let currentAccessToken = ''
let authRefreshPromise = null

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

function tokenExpiresAt(token) {
  const claims = decodeJWTClaims(token)
  return claims.exp ? claims.exp * 1000 : 0
}

function shouldRefreshAccessToken(session) {
  if (!session?.accessToken || !session?.refreshToken) {
    return false
  }
  const expiresAt = tokenExpiresAt(session.accessToken) || session.expiresAt || 0
  return expiresAt <= Date.now() + 60000
}

async function refreshAuthSession(force = false) {
  if (typeof window === 'undefined') {
    return null
  }

  const session = loadAuthSession({ allowExpiredAccessToken: true })
  if (!session?.refreshToken) {
    return null
  }
  if (!force && !shouldRefreshAccessToken(session)) {
    return session
  }
  if (authRefreshPromise) {
    return authRefreshPromise
  }

  authRefreshPromise = fetch('/api/auth/token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken: session.refreshToken }),
  })
    .then(async (response) => {
      const payload = await response.json().catch(() => ({}))
      if (!response.ok || !payload.access_token) {
        throw new Error(payload.error_description || payload.error || 'Keycloak refresh failed')
      }

      const nextSession = {
        ...session,
        accessToken: payload.access_token,
        idToken: payload.id_token || session.idToken,
        refreshToken: payload.refresh_token || session.refreshToken,
        expiresAt: Date.now() + Number(payload.expires_in || 0) * 1000,
      }
      saveAuthSession(nextSession)
      window.dispatchEvent(new CustomEvent('alemlive-auth-refreshed', { detail: nextSession }))
      return nextSession
    })
    .catch((error) => {
      clearAuthSession()
      throw error
    })
    .finally(() => {
      authRefreshPromise = null
    })

  return authRefreshPromise
}

async function apiRequest(path, options = {}) {
  const { requireAuth = false, skipAuthRefresh = false, retryOnAuth = true, ...requestOptions } = options
  if (!skipAuthRefresh) {
    await refreshAuthSession(false).catch(() => null)
  }

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
      if (!skipAuthRefresh && retryOnAuth) {
        const refreshed = await refreshAuthSession(true).catch(() => null)
        if (refreshed?.accessToken) {
          return apiRequest(path, { ...options, retryOnAuth: false })
        }
      }
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

function loadAuthSession(options = {}) {
  const { allowExpiredAccessToken = false } = options
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const session = JSON.parse(window.localStorage.getItem(authSessionKey))
    if (!session?.accessToken || (!allowExpiredAccessToken && isJWTExpired(session.accessToken) && !session.refreshToken)) {
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

function MediaStateReporter({ roomName }) {
  const { isMicrophoneEnabled, isCameraEnabled } = useLocalParticipant()

  function reportDeviceState(device, enabled) {
    if (!roomName) {
      return
    }
    apiRequest(`/api/rooms/${encodeURIComponent(roomName)}/device-state`, {
      method: 'POST',
      body: JSON.stringify({ device, enabled }),
    }).catch(() => {})
  }

  useEffect(() => {
    reportDeviceState('mic', Boolean(isMicrophoneEnabled))
  }, [roomName, isMicrophoneEnabled])

  useEffect(() => {
    reportDeviceState('camera', Boolean(isCameraEnabled))
  }, [roomName, isCameraEnabled])

  return null
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
  const [selectedReportId, setSelectedReportId] = useState(initialReportId || '')
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
  const [reports, setReports] = useState([])
  const [reportFilters, setReportFilters] = useState(null)
  const [reportDetails, setReportDetails] = useState({})
  const [reportActions, setReportActions] = useState({})
  const [personalNotes, setPersonalNotes] = useState({})
  const [isPersonalNoteSaving, setIsPersonalNoteSaving] = useState(false)
  const [isPersonalizingSummary, setIsPersonalizingSummary] = useState(false)
  const [copilotLanguage, setCopilotLanguage] = useState('ru')
  const [collapsedTranscriptChapters, setCollapsedTranscriptChapters] = useState({})
  const [transcriptSearchQuery, setTranscriptSearchQuery] = useState('')
  const [transcriptMatchIndex, setTranscriptMatchIndex] = useState(0)
  const [participationSortDescending, setParticipationSortDescending] = useState(true)
  const [selectedPositionsParticipant, setSelectedPositionsParticipant] = useState('')
  const [isEditedNoticeDismissed, setIsEditedNoticeDismissed] = useState(false)
  const [showNotesChapterDescriptions, setShowNotesChapterDescriptions] = useState(true)
  const transcriptLineRefs = useRef({})
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
  const [copilotPanelWidth, setCopilotPanelWidth] = useState(420)
  const [isWideEnoughToResizeCopilot, setIsWideEnoughToResizeCopilot] = useState(() => (
    typeof window === 'undefined' || window.innerWidth > 1320
  ))
  const isResizingCopilotRef = useRef(false)
  const [reportMirrorOverrides, setReportMirrorOverrides] = useState({})
  const [roomSettings, setRoomSettings] = useState(null)
  const [isRoomSettingsOpen, setIsRoomSettingsOpen] = useState(false)
  const [roomRecordingStatus, setRoomRecordingStatus] = useState({ state: 'idle', configured: false })
  const [isRecordingToggling, setIsRecordingToggling] = useState(false)
  const copilotInputRef = useRef(null)
  const recordingVideoRef = useRef(null)
  const thumbnailVideoRef = useRef(null)
  const [highlightThumbnails, setHighlightThumbnails] = useState({})
  const [chapterThumbnails, setChapterThumbnails] = useState({})
  const reportUploadInputRef = useRef(null)

  const canStart = form.userName.trim() && form.roomName.trim()
  const isConnected = Boolean(meeting)
  const isAuthEnabled = Boolean(authConfig.enabled)
  const isAuthenticated = !isAuthEnabled || Boolean(authSession?.accessToken)
  const selectedReportDetail = reportDetails[selectedReportId]
  const selectedReport = selectedReportDetail?.report || reports.find((report) => report.id === selectedReportId) || reports[0] || reportRows[0]
  const searchableTranscriptLines = selectedReportDetail?.transcriptLines || selectedReportDetail?.transcript || transcriptLines
  const searchableChapters = selectedReportDetail?.chapters || chapters
  const searchableHighlights = selectedReportDetail?.highlights || highlights
  const selectedReportRecordingUrl = selectedReportDetail?.recordingUrl || ''
  const transcriptChapterGroups = groupTranscriptByChapters(searchableTranscriptLines, searchableChapters)
  const transcriptMatches = findTranscriptMatches(transcriptChapterGroups, transcriptSearchQuery)
  const normalizedTranscriptMatchIndex = transcriptMatches.length > 0
    ? ((transcriptMatchIndex % transcriptMatches.length) + transcriptMatches.length) % transcriptMatches.length
    : 0
  const activeTranscriptMatch = transcriptMatches.length > 0 ? transcriptMatches[normalizedTranscriptMatchIndex] : null

  useEffect(() => {
    if (!activeTranscriptMatch) {
      return
    }
    setCollapsedTranscriptChapters((current) => {
      if (!current[activeTranscriptMatch.chapterKey]) {
        return current
      }
      const next = { ...current }
      delete next[activeTranscriptMatch.chapterKey]
      return next
    })
  }, [activeTranscriptMatch?.lineKey, activeTranscriptMatch?.chapterKey])

  useEffect(() => {
    if (!activeTranscriptMatch) {
      return
    }
    const node = transcriptLineRefs.current[activeTranscriptMatch.lineKey]
    node?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [activeTranscriptMatch?.lineKey, transcriptMatchIndex, collapsedTranscriptChapters])

  useEffect(() => {
    if (activeReportTab !== 'highlights' || !selectedReportRecordingUrl) {
      return undefined
    }
    const video = thumbnailVideoRef.current
    if (!video) {
      return undefined
    }

    let cancelled = false

    async function run() {
      if (video.readyState < 1) {
        await new Promise((resolve) => {
          video.addEventListener('loadedmetadata', resolve, { once: true })
        })
      }
      for (const item of searchableHighlights) {
        if (cancelled) {
          return
        }
        const key = highlightKey(item)
        if (highlightThumbnails[key]) {
          continue
        }
        try {
          const dataUrl = await captureVideoFrame(video, timeToSeconds(item.time))
          if (!cancelled) {
            setHighlightThumbnails((current) => ({ ...current, [key]: dataUrl }))
          }
        } catch {
          // Skip thumbnails that fail to capture (e.g. seek target out of range).
        }
      }
    }

    run()

    return () => {
      cancelled = true
    }
  }, [activeReportTab, selectedReportRecordingUrl, searchableHighlights])

  useEffect(() => {
    if (activeReportTab !== 'chapters' || !selectedReportRecordingUrl) {
      return undefined
    }
    const video = thumbnailVideoRef.current
    if (!video) {
      return undefined
    }

    let cancelled = false

    async function run() {
      if (video.readyState < 1) {
        await new Promise((resolve) => {
          video.addEventListener('loadedmetadata', resolve, { once: true })
        })
      }
      for (const item of searchableChapters) {
        if (cancelled) {
          return
        }
        const key = chapterKey(item)
        if (chapterThumbnails[key]) {
          continue
        }
        try {
          const dataUrl = await captureVideoFrame(video, timeToSeconds(item.start || item.time))
          if (!cancelled) {
            setChapterThumbnails((current) => ({ ...current, [key]: dataUrl }))
          }
        } catch {
          // Skip thumbnails that fail to capture (e.g. seek target out of range).
        }
      }
    }

    run()

    return () => {
      cancelled = true
    }
  }, [activeReportTab, selectedReportRecordingUrl, searchableChapters])

  useEffect(() => {
    function handleMouseMove(event) {
      if (!isResizingCopilotRef.current) {
        return
      }
      const nextWidth = window.innerWidth - event.clientX
      setCopilotPanelWidth(Math.min(640, Math.max(320, nextWidth)))
    }
    function handleMouseUp() {
      if (!isResizingCopilotRef.current) {
        return
      }
      isResizingCopilotRef.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
    return () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }
  }, [])

  useEffect(() => {
    function handleResize() {
      setIsWideEnoughToResizeCopilot(window.innerWidth > 1320)
    }
    window.addEventListener('resize', handleResize)
    return () => window.removeEventListener('resize', handleResize)
  }, [])

  function startCopilotResize(event) {
    event.preventDefault()
    isResizingCopilotRef.current = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }

  const dateFilterOptions = quickDateOptions.map((option) => ({
    ...option,
    label: reportFilters?.quickDateOptions?.find((backendOption) => backendOption.id === option.id)?.label || option.label,
  }))
  const activeQuickDateOption = dateFilterOptions.find((option) => option.id === timeFilterMode)
  const timeFilterLabel = timeFilterMode === 'custom' ? formatDateRange(timeFilterRange) : activeQuickDateOption?.label || quickDateOptions[0].label
  const calendarDays = getCalendarDays(calendarMonth)
  const areAllTypeFiltersSelected = selectedTypeFilterIds.length === typeFilterOptions.length
  const visibleReports = filterReportsByType(reports, selectedTypeFilterIds)
  const hasProcessingReports = reports.some((report) => ['processing', 'recording'].includes(report.processingState || report.status))
  const currentRoomRecordingState = roomRecordingStatus?.state || roomRecordingStatus?.status || 'idle'
  const isRoomRecording = currentRoomRecordingState === 'recording'
  const selectedReportMirrorCorrection = reportMirrorOverrides[selectedReportId] ?? Boolean(selectedReportDetail?.recordingMirrorCorrection)
  const selectedReportRecordingMessage = (() => {
    const state = selectedReport?.processingState || selectedReport?.status || ''
    if (state === 'recording') {
      return 'Запись еще идет. Видео появится после завершения LiveKit Egress и обработки отчета.'
    }
    if (state === 'processing') {
      return 'Отчет обрабатывается. Видео появится здесь, когда backend сохранит файл записи.'
    }
    return 'Для этой встречи нет сохраненного видео. Включите LiveKit Egress и storage, чтобы записи автоматически появлялись в отчетах.'
  })()

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
          setReports([])
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
    if (!authReady || !isAuthenticated || activeView !== 'reportDetail' || !selectedReportId) {
      return undefined
    }

    let isMounted = true

    apiRequest(`/api/reports/${selectedReportId}/personal-note`)
      .then((payload) => {
        if (isMounted) {
          setPersonalNotes((current) => ({ ...current, [selectedReportId]: payload.note || '' }))
        }
      })
      .catch(() => {
        // Personal note is optional; keep the field empty if it cannot be loaded.
      })

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
    if (!isConnected || !meetingMeta.room) {
      setRoomRecordingStatus({ state: 'idle', configured: false })
      return undefined
    }

    refreshRoomRecordingStatus(meetingMeta.room)
    const timer = window.setInterval(() => {
      refreshRoomRecordingStatus(meetingMeta.room)
    }, 5000)

    return () => window.clearInterval(timer)
  }, [isConnected, meetingMeta.room])

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
    function handleAuthRefreshed(event) {
      if (event.detail?.accessToken) {
        setAuthSession(event.detail)
        setAuthError('')
      }
    }

    window.addEventListener('alemlive-auth-expired', handleAuthExpired)
    window.addEventListener('alemlive-auth-refreshed', handleAuthRefreshed)
    return () => {
      window.removeEventListener('alemlive-auth-expired', handleAuthExpired)
      window.removeEventListener('alemlive-auth-refreshed', handleAuthRefreshed)
    }
  }, [])

  useEffect(() => {
    if (!authReady || !authSession?.refreshToken) {
      return undefined
    }

    let isMounted = true
    const refresh = () => {
      refreshAuthSession(false)
        .then((nextSession) => {
          if (isMounted && nextSession?.accessToken) {
            setAuthSession(nextSession)
            setAuthError('')
          }
        })
        .catch(() => {
          if (isMounted) {
            setAuthSession(null)
            setAuthError('Сессия истекла. Войдите заново.')
          }
        })
    }

    refresh()
    const timer = window.setInterval(refresh, 30000)
    return () => {
      isMounted = false
      window.clearInterval(timer)
    }
  }, [authReady, authSession?.refreshToken, authSession?.accessToken])

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

  async function refreshRoomRecordingStatus(roomName = meetingMeta.room) {
    if (!roomName) {
      return null
    }

    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(roomName)}/recording/status`).catch(() => null)
    if (payload) {
      setRoomRecordingStatus(payload)
    }
    return payload
  }

  async function toggleRoomRecording() {
    if (!meetingMeta.room || isRecordingToggling) {
      return
    }

    setIsRecordingToggling(true)
    setMeetingNotice('')

    try {
      const action = isRoomRecording ? 'stop' : 'start'
      const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/recording/${action}`, {
        method: 'POST',
      })
      setRoomRecordingStatus(payload)
      setWorkspaceNotice(isRoomRecording ? 'Запись останавливается, отчёт появится после обработки' : 'Запись конференции началась')
      if (payload?.reportId) {
        setReportsRefreshKey((current) => current + 1)
      }
    } catch (error) {
      setMeetingNotice(error.message || 'Не удалось изменить статус записи')
    } finally {
      setIsRecordingToggling(false)
    }
  }

  async function requestToken(roomName, userName, isHost) {
    const payload = await apiRequest('/api/livekit/token', {
      method: 'POST',
      body: JSON.stringify({ roomName, userName, isHost }),
    })

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
    const payload = await apiRequest(`/api/rooms/${encodeURIComponent(meetingMeta.room)}/leave`, {
      method: 'POST',
      body: JSON.stringify({
        userName: meetingMeta.name,
        event: 'left',
      }),
    }).catch(() => null)
    if (payload?.reportId) {
      setSelectedReportId(payload.reportId)
      setReportsRefreshKey((current) => current + 1)
      apiRequest(`/api/reports/${payload.reportId}`).then((detail) => {
        if (detail?.report) {
          setReportDetails((current) => ({ ...current, [payload.reportId]: detail }))
          setReports((current) => [detail.report, ...current.filter((report) => report.id !== detail.report.id)])
        }
      }).catch(() => null)
      setWorkspaceNotice(`Отчет встречи сохранен: ${payload.reportId}`)
    }
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
      if (payload.available && payload.url) {
        window.open(payload.url, '_blank', 'noopener,noreferrer')
      }
      setReportActionMessage(payload.available
        ? `Запись: ${payload.duration}, маркеры ${payload.markers?.join(', ') || 'нет'}`
        : payload.message || 'Запись пока недоступна')
    }
  }

  function seekRecordingTo(timeLabel) {
    const node = recordingVideoRef.current
    if (!node || !timeLabel) {
      return
    }
    node.currentTime = timeToSeconds(timeLabel)
    node.play().catch(() => {})
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

  function toggleReportMirrorCorrection() {
    if (!selectedReportId) {
      return
    }

    setReportMirrorOverrides((current) => ({
      ...current,
      [selectedReportId]: !(current[selectedReportId] ?? Boolean(selectedReportDetail?.recordingMirrorCorrection)),
    }))
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
      const currentSummary = selectedReportDetail?.summary?.length
        ? selectedReportDetail.summary
        : [{ title: selectedReport.title || 'Сводка', text: '' }]
      const currentActionItems = selectedReportDetail?.actionItems || []
      const title = window.prompt('Заголовок сводки', currentSummary[0]?.title || '')
      if (title === null) {
        return
      }

      const text = window.prompt('Текст сводки', currentSummary[0]?.text || '')
      if (text === null) {
        return
      }

      const nextSummary = [
        {
          ...currentSummary[0],
          title: title.trim() || currentSummary[0]?.title || 'Сводка',
          text: text.trim() || currentSummary[0]?.text || '',
        },
        ...currentSummary.slice(1),
      ]

      const payload = await apiRequest(`/api/reports/${selectedReportId}/notes`, {
        method: 'PATCH',
        body: JSON.stringify({
          summary: nextSummary,
          actionItems: currentActionItems,
        }),
      })
      setReportDetails((current) => ({
        ...current,
        [selectedReportId]: {
          ...(current[selectedReportId] || selectedReportDetail || {}),
          summary: payload.summary || nextSummary,
          actionItems: payload.actionItems || currentActionItems,
        },
      }))
      setReportActionMessage('Заметки сохранены')
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
        body: JSON.stringify({ message: text, language: copilotLanguage }),
      })
      setCopilotMessages((current) => [...current, { role: 'assistant', text: payload.answer || 'Ответ пустой' }])
    } catch (error) {
      setCopilotMessages((current) => [...current, { role: 'assistant', text: error.message }])
    } finally {
      setIsCopilotSending(false)
    }
  }

  function askAboutMoment(question, time) {
    if (time) {
      seekRecordingTo(time)
    }
    focusCopilotPanel()
    askReportCopilot(question)
  }

  async function savePersonalNote(reportId, note) {
    if (!reportId) {
      return
    }

    setIsPersonalNoteSaving(true)
    try {
      await apiRequest(`/api/reports/${reportId}/personal-note`, {
        method: 'PUT',
        body: JSON.stringify({ note }),
      })
    } catch {
      // Keep the locally typed note even if saving failed; the next edit will retry.
    } finally {
      setIsPersonalNoteSaving(false)
    }
  }

  async function personalizeSummary() {
    if (!selectedReportId || isPersonalizingSummary) {
      return
    }

    setIsPersonalizingSummary(true)
    try {
      const payload = await apiRequest(`/api/reports/${selectedReportId}/personalize`, {
        method: 'POST',
      })
      setReportDetails((current) => ({
        ...current,
        [selectedReportId]: {
          ...(current[selectedReportId] || selectedReportDetail || {}),
          summary: payload.summary,
        },
      }))
    } catch (error) {
      setReportActionMessage(error.message)
    } finally {
      setIsPersonalizingSummary(false)
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
              className={isRoomRecording ? 'recording-control active' : 'recording-control'}
              type="button"
              onClick={toggleRoomRecording}
              disabled={isRecordingToggling || currentRoomRecordingState === 'processing'}
              aria-label={isRoomRecording ? 'Остановить запись конференции' : 'Начать запись конференции'}
              aria-pressed={isRoomRecording}
              title={roomRecordingLabel(currentRoomRecordingState)}
            >
              {isRecordingToggling ? <Loader2 className="spin-icon" size={18} /> : isRoomRecording ? <Square size={16} fill="currentColor" /> : <Radio size={18} />}
              <span>{isRoomRecording ? 'Остановить' : 'Запись'}</span>
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
              <MediaStateReporter roomName={meeting.roomName} />
              <div className="meeting-livekit-layout">
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
              </div>
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
              className={[
                'report-row',
                selectedReportId === report.id ? 'selected' : '',
                openReportActionsId === report.id ? 'actions-open' : '',
              ].filter(Boolean).join(' ')}
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
                      {normalizeReportActions(reportActions[report.id]).map((action) => (
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
    const detailKeyQuestions = selectedReportDetail?.keyQuestions || []

    if (activeReportTab === 'notes') {
      return (
        <div className="detail-notes">
          <div className="score-strip">
            <div>
              <span>Оценка Alem</span>
              <strong>{selectedReport.score}</strong>
              <small>{scoreLabel(selectedReport.score)}</small>
            </div>
            <div>
              <span>Вовлечённость</span>
              <strong>{selectedReport.engagementScore}</strong>
              <small>{scoreLabel(selectedReport.engagementScore)}</small>
            </div>
            <div>
              <span>Настроение</span>
              <strong>{selectedReport.moodScore}</strong>
              <small>{scoreLabel(selectedReport.moodScore)}</small>
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
            <button
              className="personalize-summary-button"
              type="button"
              onClick={personalizeSummary}
              disabled={isPersonalizingSummary}
            >
              <Sparkles size={16} />
              {isPersonalizingSummary ? 'Персонализируем...' : 'Персонализировать резюме'}
            </button>
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
                    {item.time && <time className="action-item-time">{item.time}</time>}
                    <h4>{item.task || item.title}</h4>
                    <p>{[item.owner, item.due].filter(Boolean).join(' · ')}</p>
                    <button
                      type="button"
                      className="ask-alem-link"
                      onClick={() => askAboutMoment(`Расскажи подробнее про задачу: ${item.task || item.title}`, item.time)}
                    >
                      Спросить Alem
                    </button>
                  </div>
                </article>
              ))}
            </div>
          </section>

          {detailKeyQuestions.length > 0 && (
            <section className="key-questions-panel">
              <div className="section-kicker">
                <HelpCircle size={18} />
                Ключевые вопросы
              </div>
              <div className="key-questions-list">
                {detailKeyQuestions.map((item, index) => (
                  <article className="key-question-item" key={`${item.question}-${index}`}>
                    <div className="key-question-head">
                      {item.time && <time>{item.time}</time>}
                      <p>{item.question}</p>
                    </div>
                    <button
                      type="button"
                      className="ask-alem-link"
                      onClick={() => askAboutMoment(item.question, item.time)}
                    >
                      Спросить Alem
                    </button>
                    {item.answer && (
                      <div className="key-question-answer">
                        <Sparkles size={14} />
                        <span><strong>Предложенный ответ:</strong> {item.answer}</span>
                      </div>
                    )}
                  </article>
                ))}
              </div>
            </section>
          )}

          {detailChapters.length > 0 && (
            <section className="notes-chapters-panel">
              <div className="notes-chapters-header">
                <div className="section-kicker">
                  <ListChecks size={18} />
                  Главы и темы
                </div>
                <label className="chapter-description-toggle">
                  Описания
                  <button
                    type="button"
                    className={showNotesChapterDescriptions ? 'toggle-switch on' : 'toggle-switch'}
                    onClick={() => setShowNotesChapterDescriptions((current) => !current)}
                    aria-pressed={showNotesChapterDescriptions}
                    aria-label="Показать описания глав"
                  >
                    <span />
                  </button>
                </label>
              </div>
              <div className="notes-chapters-list">
                {detailChapters.map((chapterItem) => (
                  <article className="notes-chapter-item" key={chapterKey(chapterItem)}>
                    <button
                      type="button"
                      className="notes-chapter-title"
                      onClick={() => seekRecordingTo(chapterItem.start || chapterItem.time)}
                    >
                      <time>{chapterItem.time || chapterItem.start}</time>
                      <h4>{chapterItem.title}</h4>
                    </button>
                    {showNotesChapterDescriptions && (chapterItem.text || chapterItem.points?.length > 0) && (
                      <div className="notes-chapter-description">
                        {chapterItem.text && <p>{chapterItem.text}</p>}
                        {chapterItem.points?.length > 0 && (
                          <ul>
                            {chapterItem.points.map((point, pointIndex) => (
                              <li key={pointIndex}>{point}</li>
                            ))}
                          </ul>
                        )}
                      </div>
                    )}
                  </article>
                ))}
              </div>
            </section>
          )}

          <section className="personal-notes-panel">
            <div className="section-kicker">
              <Edit3 size={18} />
              Ваши заметки
            </div>
            <span className="private-pill">Только вы можете это видеть</span>
            <textarea
              className="personal-note-input"
              value={personalNotes[selectedReportId] || ''}
              placeholder="Добавьте личную заметку об этой встрече..."
              onChange={(event) => setPersonalNotes((current) => ({ ...current, [selectedReportId]: event.target.value }))}
              onBlur={(event) => savePersonalNote(selectedReportId, event.target.value)}
            />
            {isPersonalNoteSaving && <span className="personal-note-status">Сохраняем...</span>}
          </section>
        </div>
      )
    }

    if (activeReportTab === 'transcript') {
      const detailKeywords = selectedReportDetail?.keywords || []
      const hasSearchQuery = transcriptSearchQuery.trim().length > 0
      const totalMatches = transcriptMatches.length

      return (
        <div className="transcript-report report-pane">
          <div className="transcript-tools">
            <div className="transcript-search">
              <Search size={18} />
              <input
                value={transcriptSearchQuery}
                onChange={(event) => {
                  setTranscriptSearchQuery(event.target.value)
                  setTranscriptMatchIndex(0)
                }}
                placeholder="Поиск по транскрипту: token, комната, отчёт"
              />
              {hasSearchQuery && (
                <div className="transcript-search-nav">
                  <span className="transcript-search-count">
                    {totalMatches > 0 ? `${normalizedTranscriptMatchIndex + 1} из ${totalMatches}` : '0 из 0'}
                  </span>
                  <button
                    type="button"
                    className="icon-button"
                    disabled={totalMatches === 0}
                    onClick={() => setTranscriptMatchIndex((current) => current - 1)}
                    aria-label="Предыдущее совпадение"
                  >
                    <ChevronUp size={16} />
                  </button>
                  <button
                    type="button"
                    className="icon-button"
                    disabled={totalMatches === 0}
                    onClick={() => setTranscriptMatchIndex((current) => current + 1)}
                    aria-label="Следующее совпадение"
                  >
                    <ChevronDown size={16} />
                  </button>
                  <button
                    type="button"
                    className="icon-button"
                    onClick={() => {
                      setTranscriptSearchQuery('')
                      setTranscriptMatchIndex(0)
                    }}
                    aria-label="Очистить поиск"
                  >
                    <X size={16} />
                  </button>
                </div>
              )}
            </div>
            <span className="report-badge muted">{detailTranscriptLines.length} моментов</span>
          </div>

          {detailKeywords.length > 0 && (
            <div className="transcript-keywords">
              <span className="transcript-keywords-label">Ключевые слова</span>
              <div className="keyword-chip-row">
                {detailKeywords.map((word) => (
                  <span
                    className="keyword-chip"
                    key={word}
                    role="button"
                    tabIndex={0}
                    onClick={() => {
                      setTranscriptSearchQuery(word)
                      setTranscriptMatchIndex(0)
                    }}
                  >
                    {word}
                  </span>
                ))}
              </div>
            </div>
          )}

          <div className="transcript-list transcript-list-grouped">
            {transcriptChapterGroups.map((group, groupIndex) => {
              const chapterKey = group.chapter?.title || `untitled-${groupIndex}`
              const isCollapsed = Boolean(collapsedTranscriptChapters[chapterKey])

              return (
                <section className="transcript-chapter-group" key={chapterKey}>
                  {group.chapter && (
                    <button
                      type="button"
                      className="transcript-chapter-header"
                      onClick={() => setCollapsedTranscriptChapters((current) => ({ ...current, [chapterKey]: !current[chapterKey] }))}
                      aria-expanded={!isCollapsed}
                    >
                      <span>{group.chapter.title}</span>
                      {isCollapsed ? <ChevronDown size={18} /> : <ChevronUp size={18} />}
                    </button>
                  )}
                  {!isCollapsed && (
                    <div className="transcript-bubble-list">
                      {group.lines.map((line) => {
                        const lineKey = transcriptLineKey(line)
                        const isActiveLine = activeTranscriptMatch?.lineKey === lineKey
                        const segments = hasSearchQuery
                          ? splitTextForHighlight(line.text, transcriptSearchQuery, isActiveLine ? activeTranscriptMatch.charIndex : -1)
                          : null

                        return (
                          <article
                            className={isActiveLine ? 'transcript-bubble active-match' : 'transcript-bubble'}
                            key={lineKey}
                            ref={(node) => { transcriptLineRefs.current[lineKey] = node }}
                            role="button"
                            tabIndex={0}
                            onClick={() => seekRecordingTo(line.time)}
                            onKeyDown={(event) => {
                              if (event.key === 'Enter' || event.key === ' ') {
                                event.preventDefault()
                                seekRecordingTo(line.time)
                              }
                            }}
                            aria-label={`Перейти к моменту ${line.time}, где говорит ${line.speaker}`}
                          >
                            <span className="transcript-avatar" style={{ background: speakerAvatarColor(line.speaker) }}>
                              {speakerInitials(line.speaker)}
                            </span>
                            <div className="transcript-bubble-body">
                              <div className="transcript-bubble-meta">
                                <strong>{line.speaker}</strong>
                                <time>{line.time}</time>
                              </div>
                              <p className={line.sentiment ? `transcript-bubble-text sentiment-${line.sentiment}` : 'transcript-bubble-text'}>
                                {segments
                                  ? segments.map((segment, segmentIndex) => (segment.match ? (
                                    <mark className={segment.active ? 'search-hit active' : 'search-hit'} key={segmentIndex}>
                                      {segment.text}
                                    </mark>
                                  ) : (
                                    <span key={segmentIndex}>{segment.text}</span>
                                  )))
                                  : line.text}
                              </p>
                              {line.sentiment && (
                                <span className={`sentiment-tag sentiment-tag-${line.sentiment}`}>
                                  {line.sentiment === 'positive' ? 'Повышено Sentiment' : 'Снижено Sentiment'}
                                </span>
                              )}
                            </div>
                          </article>
                        )
                      })}
                    </div>
                  )}
                </section>
              )
            })}
          </div>
        </div>
      )
    }

    if (activeReportTab === 'deepDive') {
      const moodScore = selectedReport.moodScore ?? 0
      const engagementScore = selectedReport.engagementScore ?? 0
      const interruptionsCount = selectedReportDetail?.interruptions ?? 0
      const trendPoints = selectedReportDetail?.trend || []
      const sortedSpeakers = [...detailSpeakerStats].sort((a, b) => {
        const aValue = a.talk ?? a.talkTime ?? 0
        const bValue = b.talk ?? b.talkTime ?? 0
        return participationSortDescending ? bValue - aValue : aValue - bValue
      })
      const positionsParticipant = detailSpeakerStats.find((speaker) => speaker.name === selectedPositionsParticipant) || sortedSpeakers[0]
      const charismaSpeakers = detailSpeakerStats.filter((speaker) => speaker.hasCharisma)
      const biasSpeakers = detailSpeakerStats.filter((speaker) => speaker.hasBias)
      const positionAverages = {
        mic: averageOf(detailSpeakerStats.map((speaker) => speaker.micMutedPercent ?? 0)),
        camera: averageOf(detailSpeakerStats.map((speaker) => speaker.cameraOffPercent ?? 0)),
        score: averageOf(detailSpeakerStats.map((speaker) => speaker.score ?? 0)),
        mood: averageOf(detailSpeakerStats.map((speaker) => speaker.mood ?? 0)),
        engagement: averageOf(detailSpeakerStats.map((speaker) => speaker.engagement ?? 0)),
        charisma: averageOf(charismaSpeakers.map((speaker) => speaker.charisma)),
        bias: averageOf(biasSpeakers.map((speaker) => speaker.bias)),
      }

      return (
        <div className="deep-report report-pane">
          {!isEditedNoticeDismissed && (
            <div className="edited-notice">
              <Info size={18} />
              <div>
                <strong>Этот отчёт был отредактирован</strong>
                <p>Этот вид основан на оригинальной транскрипции и не может отражать последующие изменения.</p>
              </div>
              <button type="button" className="icon-button" onClick={() => setIsEditedNoticeDismissed(true)} aria-label="Скрыть уведомление">
                <X size={18} />
              </button>
            </div>
          )}

          <section className="participation-panel">
            <div className="participation-header">
              <span className="section-kicker-plain">УЧАСТИЕ</span>
              <button
                type="button"
                className="sort-toggle-button"
                onClick={() => setParticipationSortDescending((current) => !current)}
              >
                <ArrowDownUp size={16} />
                Сортировать по времени выступления
                <ChevronDown size={16} />
              </button>
            </div>
            <div className="participation-list">
              {sortedSpeakers.map((speaker) => {
                const talkValue = speaker.talk ?? speaker.talkTime ?? 0
                const isSelected = positionsParticipant?.name === speaker.name
                return (
                  <article
                    className={isSelected ? 'participation-row selected' : 'participation-row'}
                    key={speaker.name}
                    onClick={() => setSelectedPositionsParticipant(speaker.name)}
                  >
                    <span className="participation-name">
                      <Clock3 size={16} />
                      {speaker.name}
                    </span>
                    <button
                      type="button"
                      className="icon-button participation-play"
                      onClick={(event) => {
                        event.stopPropagation()
                        seekRecordingTo(speaker.firstSeenTime)
                      }}
                      disabled={!speaker.firstSeenTime}
                      aria-label={`Перейти к моменту, где говорит ${speaker.name}`}
                    >
                      <Play size={15} fill="currentColor" />
                    </button>
                    <div className="talk-bar" aria-label={`${speaker.name} говорил ${talkValue}% времени`}>
                      <span style={{ width: `${talkValue}%` }} />
                    </div>
                    <b>{talkValue}%</b>
                  </article>
                )
              })}
            </div>
          </section>

          {positionsParticipant && (
            <section className="positions-panel">
              <div className="participation-header">
                <span className="section-kicker-plain">ПОЛОЖЕНИЯ</span>
                <span className="positions-subject">{positionsParticipant.name}</span>
              </div>
              <PositionSlider
                label="Микрофон выключен"
                description="Процент времени встречи, когда микрофон был выключен"
                value={positionsParticipant.micMutedPercent ?? 0}
                hasValue
                average={positionAverages.mic}
                min={0}
                max={100}
                unit=" %"
              />
              <PositionSlider
                label="Камера отключена"
                description="Процент времени встречи, когда камера была выключена"
                value={positionsParticipant.cameraOffPercent ?? 0}
                hasValue
                average={positionAverages.camera}
                min={0}
                max={100}
                unit=" %"
              />
              <PositionSlider
                label="Оценка участника"
                description="Индивидуальный показатель на основе настроения, вовлечённости и доли участия в разговоре"
                value={positionsParticipant.score ?? 0}
                hasValue
                average={positionAverages.score}
                min={50}
                max={100}
                unit=""
              />
              <PositionSlider
                label="Настроение"
                description="Отношение к встрече на основе sentiment-меток собственных реплик"
                value={positionsParticipant.mood ?? 0}
                hasValue
                average={positionAverages.mood}
                min={50}
                max={100}
                unit=""
              />
              <PositionSlider
                label="Вовлечённость"
                description="Доля участия в разговоре и количество реплик"
                value={positionsParticipant.engagement ?? 0}
                hasValue
                average={positionAverages.engagement}
                min={50}
                max={100}
                unit=""
              />
              <PositionSlider
                label="Харизма"
                description="Насколько позитивно или негативно реагировали другие сразу после реплик этого участника"
                value={positionsParticipant.charisma ?? 0}
                hasValue={Boolean(positionsParticipant.hasCharisma)}
                average={positionAverages.charisma}
                min={50}
                max={100}
                unit=""
              />
              <PositionSlider
                label="Предвзятость"
                description="Насколько позитивно или негативно этот участник реагировал сразу после реплик других"
                value={positionsParticipant.bias ?? 0}
                hasValue={Boolean(positionsParticipant.hasBias)}
                average={positionAverages.bias}
                min={50}
                max={100}
                unit=""
              />
            </section>
          )}

          <div className="metric-grid">
            <div className="metric-card">
              <TrendingUp size={20} />
              <span>Sentiment</span>
              <strong>{moodScore}%</strong>
              <p>{moodDescription(moodScore)}</p>
            </div>
            <div className="metric-card">
              <Zap size={20} />
              <span>Engagement</span>
              <strong>{engagementScore}%</strong>
              <p>{engagementDescription(engagementScore)}</p>
            </div>
            <div className="metric-card">
              <Clock3 size={20} />
              <span>Interruptions</span>
              <strong>{interruptionsCount}</strong>
              <p>{interruptionsDescription(interruptionsCount)}</p>
            </div>
          </div>

          <section className="trend-panel">
            <div className="trend-legend">
              <span><i style={{ background: trendLineColors.score }} />Оценка Alem {selectedReport.score}</span>
              <span><i style={{ background: trendLineColors.engagement }} />Вовлечённость {engagementScore}</span>
              <span><i style={{ background: trendLineColors.mood }} />Настроение {moodScore}</span>
            </div>
            {trendPoints.length > 1 ? (
              <>
                <TrendChart points={trendPoints} />
                <div className="trend-axis">
                  {trendPoints.map((point) => <span key={point.time}>{point.time}</span>)}
                </div>
              </>
            ) : (
              <p className="trend-empty">Недостаточно реплик в транскрипте, чтобы построить график динамики по времени.</p>
            )}
          </section>
        </div>
      )
    }

    if (activeReportTab === 'highlights') {
      return (
        <div className="highlights-report report-pane">
          {detailHighlights.map((item) => {
            const meta = highlightTypeMeta(item.type)
            const thumbnail = highlightThumbnails[highlightKey(item)]
            return (
              <article
                className="highlight-card"
                key={highlightKey(item)}
                role="button"
                tabIndex={0}
                onClick={() => seekRecordingTo(item.time)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault()
                    seekRecordingTo(item.time)
                  }
                }}
                aria-label={`Перейти к моменту ${item.time}: ${item.title}`}
              >
                <div className="highlight-card-body">
                  <span className={`highlight-tag ${meta.className}`}>
                    <meta.Icon size={14} />
                    {meta.label}
                  </span>
                  <span className="highlight-time">{item.time}</span>
                  <h3>{item.title}</h3>
                  {(item.note || item.text) && <p>{item.note || item.text}</p>}
                </div>
                <div className="highlight-thumb">
                  {thumbnail ? <img src={thumbnail} alt="" /> : <div className="highlight-thumb-placeholder" />}
                  <span className="highlight-thumb-play">
                    <Play size={14} fill="currentColor" />
                  </span>
                </div>
              </article>
            )
          })}
        </div>
      )
    }

    const sortedChapters = [...detailChapters].sort(
      (a, b) => timeToSeconds(a.start || a.time) - timeToSeconds(b.start || b.time),
    )

    return (
      <div className="chapters-report report-pane">
        {sortedChapters.map((chapter, index) => {
          const key = chapterKey(chapter)
          const thumbnail = chapterThumbnails[key]
          const next = sortedChapters[index + 1]
          const nextStart = next ? timeToSeconds(next.start || next.time) : null
          const durationLabel = formatDurationLabel(chapterDurationSeconds(chapter, nextStart))

          return (
            <article
              className="chapter-row"
              key={key}
              role="button"
              tabIndex={0}
              onClick={() => seekRecordingTo(chapter.start || chapter.time)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  seekRecordingTo(chapter.start || chapter.time)
                }
              }}
              aria-label={`Перейти к главе ${chapter.title}`}
            >
              <div className="chapter-row-body">
                <time>{chapter.time || chapter.start}</time>
                <h3>{chapter.title}</h3>
              </div>
              <div className="chapter-thumb">
                {thumbnail ? <img src={thumbnail} alt="" /> : <div className="chapter-thumb-placeholder" />}
                <span className="chapter-thumb-play">
                  <Play size={14} fill="currentColor" />
                </span>
                {durationLabel && <span className="chapter-thumb-duration">{durationLabel}</span>}
              </div>
            </article>
          )
        })}
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

        <div
          className="report-detail-layout"
          style={isCopilotCollapsed || !isWideEnoughToResizeCopilot ? undefined : { gridTemplateColumns: `minmax(0, 1fr) ${copilotPanelWidth}px` }}
        >
          <div className="report-recording-column">
            {selectedReportRecordingUrl && (
              <video
                ref={thumbnailVideoRef}
                key={`thumb-${selectedReportRecordingUrl}`}
                src={selectedReportRecordingUrl}
                muted
                preload="metadata"
                className="thumbnail-capture-video"
                aria-hidden="true"
              />
            )}
            {selectedReportRecordingUrl ? (
              <div className={selectedReportMirrorCorrection ? 'report-video-frame mirror-corrected' : 'report-video-frame'}>
                <video
                  ref={recordingVideoRef}
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
                <button
                  className={selectedReportMirrorCorrection ? 'video-mirror-button active' : 'video-mirror-button'}
                  type="button"
                  onClick={toggleReportMirrorCorrection}
                  aria-label={selectedReportMirrorCorrection ? 'Показать исходное видео' : 'Исправить зеркальность видео'}
                >
                  <RefreshCw size={18} />
                </button>
              </div>
            ) : (
              <div className="video-player-empty">
                <Video size={42} />
                <h3>Видео недоступно</h3>
                <p>{selectedReportRecordingMessage}</p>
                <button className="soft-action" type="button" onClick={openReportRecording}>
                  <RefreshCw size={18} />
                  Проверить запись
                </button>
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
                      <button className="detail-more-item" type="button" onClick={() => handleReportAction(selectedReport.id, 'rename')}>
                        <Edit3 size={18} />
                        Переименовать отчёт
                      </button>
                      <button className="detail-more-item" type="button" onClick={editReportNotes}>
                        <Edit3 size={18} />
                        Редактировать заметки
                      </button>
                      <button className="detail-more-item danger" type="button" onClick={() => handleReportAction(selectedReport.id, 'delete')}>
                        <Trash2 size={18} />
                        Удалить отчёт
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>

            <div className="detail-report-content">{renderReportPane()}</div>
          </div>

          <aside className={isCopilotCollapsed ? 'report-copilot collapsed' : 'report-copilot'}>
            {!isCopilotCollapsed && isWideEnoughToResizeCopilot && (
              <div
                className="copilot-resize-handle"
                onMouseDown={startCopilotResize}
                role="separator"
                aria-orientation="vertical"
                aria-label="Изменить ширину панели Alem"
              >
                <GripVertical size={14} />
              </div>
            )}
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
            <div className="copilot-scroll-area">
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
            </div>

            <form
              className="copilot-input"
              onSubmit={(event) => {
                event.preventDefault()
                askReportCopilot(copilotInput)
              }}
            >
              <span className="copilot-language-select">
                <Globe size={16} />
                <select
                  value={copilotLanguage}
                  onChange={(event) => setCopilotLanguage(event.target.value)}
                  aria-label="Язык ответа Alem"
                >
                  <option value="ru">RU</option>
                  <option value="en">EN</option>
                  <option value="kk">KK</option>
                </select>
              </span>
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
