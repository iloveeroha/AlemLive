import { useMemo, useState } from 'react'
import {
  ArrowDown,
  ArrowLeft,
  BarChart3,
  Bell,
  Bot,
  CalendarDays,
  CameraOff,
  CheckCircle2,
  ChevronDown,
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
  TrendingUp,
  Users,
  Video,
  Volume2,
  Zap,
} from 'lucide-react'
import { LiveKitRoom, VideoConference } from '@livekit/components-react'
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

function getInitialReportId() {
  if (typeof window === 'undefined') {
    return ''
  }

  const [, reportId] = window.location.hash.match(/^#report\/(.+)$/) || []
  return reportId || ''
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

  const canStart = form.userName.trim() && form.roomName.trim()
  const isConnected = Boolean(meeting)
  const selectedReport = reportRows.find((report) => report.id === selectedReportId) || reportRows[0]

  const meetingMeta = useMemo(() => {
    const room = meeting?.roomName || form.roomName || 'alem-meeting'
    const name = meeting?.userName || form.userName || 'Guest'

    return {
      room,
      name,
      initial: name.trim().slice(0, 1).toUpperCase() || 'M',
    }
  }, [form.roomName, form.userName, meeting])

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
      return next
    })
  }

  async function requestToken(roomName, userName) {
    const response = await fetch('/api/livekit/token', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ roomName, userName }),
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
      const payload = await requestToken(nextRoomName, nextUserName)

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
        audio: devices.mic,
        video: devices.camera,
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

  function leaveMeeting() {
    setMeeting(null)
  }

  async function copyRoom() {
    if (!navigator.clipboard) {
      return
    }

    await navigator.clipboard.writeText(meetingMeta.room)
  }

  function openReport(reportId) {
    setSelectedReportId(reportId)
    setActiveReportTab('notes')
    setActiveView('reportDetail')
    window.history.replaceState(null, '', `#report/${reportId}`)
  }

  function switchView(view) {
    setActiveView(view)
    if (typeof window !== 'undefined') {
      window.history.replaceState(null, '', view === 'reports' ? '#reports' : '#meeting')
    }
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
          <button className="icon-button has-dot" type="button" aria-label="Notifications">
            <Bell size={21} />
          </button>
          <button className="profile-button" type="button">
            <span className="avatar">{meetingMeta.initial}</span>
            <span>{meetingMeta.name}</span>
            <ChevronDown size={17} />
          </button>
        </div>
      </header>
    )
  }

  function renderMeetingView() {
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

          <section className="meeting-panel">
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
                <button className="icon-button" type="button" aria-label="Room link">
                  <Link size={18} />
                </button>
                <button className="icon-button" type="button" aria-label="Meeting settings">
                  <Settings size={18} />
                </button>
                {isConnected && (
                  <button className="danger-action" type="button" onClick={leaveMeeting}>
                    Завершить
                  </button>
                )}
              </div>
            </div>

            <div className={isConnected ? 'livekit-stage connected' : 'livekit-stage'}>
              {isConnected ? (
                <LiveKitRoom
                  serverUrl={meeting.serverUrl}
                  token={meeting.token}
                  connect
                  audio={meeting.audio}
                  video={meeting.video}
                  onDisconnected={leaveMeeting}
                  data-lk-theme="default"
                >
                  <VideoConference />
                </LiveKitRoom>
              ) : (
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
              )}
            </div>
          </section>

          <aside className="side-column">
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
                {['Мади Орысбек', 'Айдана Сейт', 'Team AI'].map((name, index) => (
                  <div className="member" key={name}>
                    <span className={index === 2 ? 'member-avatar ai' : 'member-avatar'}>
                      {index === 2 ? <Bot size={17} /> : name.slice(0, 1)}
                    </span>
                    <div>
                      <strong>{name}</strong>
                      <small>{index === 0 ? 'Host' : index === 1 ? 'Invited' : 'Assistant'}</small>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          </aside>
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
          <button className="ask-locale" type="button">
            <Grid2X2 size={18} />
            <ChevronDown size={16} />
          </button>
          <span>Спросите Alem о чём угодно...</span>
          <button className="ask-send" type="button" aria-label="Send question">
            <Send size={19} />
          </button>
        </div>

        <div className="reports-subnav">
          <div className="report-mode-tabs">
            <button className="report-mode active" type="button">Отчёты</button>
            <button className="report-mode" type="button">Неполный</button>
          </div>
          <div className="last-updated">
            <RefreshCw size={16} />
            Последнее обновление в 15:37
          </div>
          <button className="primary-action upload-action" type="button">
            <Download size={18} />
            Загрузить
          </button>
        </div>

        <div className="reports-filters">
          <div className="report-search-filter">
            <Search size={18} />
            <span>Фильтр по названию отчёта</span>
          </div>
          {['Все отчёты', 'В любое время', 'Тип', 'Источник', 'Папка'].map((label, index) => (
            <button className="filter-button" type="button" key={label}>
              {index === 0 ? <FileText size={17} /> : index === 1 ? <CalendarDays size={17} /> : <Filter size={17} />}
              {label}
              <ChevronDown size={16} />
            </button>
          ))}
        </div>

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

          <div className="reports-week">{reportRows[0].week}</div>

          {reportRows.map((report, index) => (
            <button
              className={index === 0 ? 'report-row selected' : 'report-row'}
              type="button"
              key={report.id}
              onClick={() => openReport(report.id)}
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
                <MoreHorizontal size={22} />
              </span>
            </button>
          ))}
        </div>
      </section>
    )
  }

  function renderReportPane() {
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
            <h3>Команда согласовала новый сценарий входа и структуру AI отчёта</h3>
            <p>
              Встреча была посвящена настройке AlemLive и аналитического отчёта после созвона. Участники договорились,
              что пользователь должен создавать комнату и присоединяться по названию, а URL и token должны подтягиваться
              автоматически через backend. После встречи агент показывает резюме, задачи, транскрипт, метрики и главы.
            </p>
          </section>

          <section className="action-list-panel">
            <div className="section-kicker">
              <CheckCircle2 size={18} />
              Action Items
            </div>
            <div className="action-items">
              {actionItems.map((item) => (
                <article className="action-item" key={item.task}>
                  <span className="action-check">
                    <CheckCircle2 size={17} />
                  </span>
                  <div>
                    <h4>{item.task}</h4>
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
            <span className="report-badge muted">3 найденных момента</span>
          </div>
          <div className="transcript-list">
            {transcriptLines.map((line) => (
              <article className="transcript-line" key={`${line.time}-${line.speaker}`}>
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
            {speakerStats.map((speaker) => (
              <article className="speaker-row" key={speaker.name}>
                <div>
                  <strong>{speaker.name}</strong>
                  <span>{speaker.sentiment} · {speaker.pace}</span>
                </div>
                <div className="talk-bar" aria-label={`${speaker.name} говорил ${speaker.talk}% времени`}>
                  <span style={{ width: `${speaker.talk}%` }} />
                </div>
                <b>{speaker.talk}%</b>
              </article>
            ))}
          </div>
        </div>
      )
    }

    if (activeReportTab === 'highlights') {
      return (
        <div className="highlights-report report-pane">
          {highlights.map((item) => (
            <article className="highlight-card" key={item.title}>
              <span className="highlight-time">{item.time}</span>
              <div>
                <h3>{item.title}</h3>
                <p>{item.note}</p>
              </div>
              <button className="icon-button" type="button" aria-label={`Open highlight ${item.title}`}>
                <Play size={17} fill="currentColor" />
              </button>
            </article>
          ))}
        </div>
      )
    }

    return (
      <div className="chapters-report report-pane">
        {chapters.map((chapter, index) => (
          <article className="chapter-row" key={chapter.title}>
            <span className="chapter-index">{index + 1}</span>
            <time>{chapter.time}</time>
            <div>
              <h3>{chapter.title}</h3>
              <p>{chapter.duration}</p>
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
                  Alison Barker, Мади, Айдана, +1 больше
                </span>
              </div>
            </div>
          </div>

          <div className="detail-actions">
            <button className="soft-action" type="button">
              <Download size={18} />
              Скачать
            </button>
            <button className="soft-action" type="button">
              <Send size={18} />
              Отправить в...
            </button>
            <button className="soft-action disabled" type="button" disabled>
              <Share2 size={18} />
              Поделиться
            </button>
          </div>
        </div>

        <div className="report-detail-layout">
          <div className="report-recording-column">
            <div className="video-player-mock">
              <div className="video-person">
                <span>{selectedReport.ownerInitial}</span>
              </div>
              <button className="video-pop-button" type="button" aria-label="Expand video">
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

            <div className="detail-tabs" role="tablist" aria-label="Разделы отчёта">
              {reportTabs.map(({ id, label, icon: Icon }) => (
                <button
                  className={activeReportTab === id ? 'detail-tab active' : 'detail-tab'}
                  type="button"
                  role="tab"
                  aria-selected={activeReportTab === id}
                  key={id}
                  onClick={() => setActiveReportTab(id)}
                >
                  <Icon size={18} />
                  {label}
                </button>
              ))}
              <button className="detail-tab icon-only" type="button" aria-label="Search in report">
                <Search size={19} />
              </button>
              <button className="detail-tab icon-only" type="button" aria-label="Copy report">
                <Copy size={19} />
              </button>
              <button className="detail-tab icon-only" type="button" aria-label="More actions">
                <MoreHorizontal size={20} />
              </button>
            </div>

            <div className="detail-report-content">{renderReportPane()}</div>
          </div>

          <aside className="report-copilot">
            <div className="copilot-tools">
              <button className="icon-button" type="button" aria-label="Edit prompts">
                <Edit3 size={18} />
              </button>
              <button className="icon-button" type="button" aria-label="History">
                <Clock3 size={18} />
              </button>
              <span />
              <button className="icon-button" type="button" aria-label="Open side panel">
                <ExternalLink size={18} />
              </button>
              <button className="icon-button" type="button" aria-label="Collapse side panel">
                <PanelRight size={18} />
              </button>
            </div>

            <div className="copilot-question-list">
              {aiQuestions.map((question) => (
                <button className="copilot-question" type="button" key={question}>
                  <Sparkles size={18} />
                  {question}
                </button>
              ))}
            </div>

            <div className="copilot-input">
              <span>Спросите Alem о чём угодно...</span>
              <button className="ask-send" type="button" aria-label="Ask Alem">
                <Send size={18} />
              </button>
            </div>
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
