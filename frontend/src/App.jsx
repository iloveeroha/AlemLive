import { useMemo, useState } from 'react'
import {
  Bell,
  Bot,
  CameraOff,
  ChevronDown,
  Contact,
  Copy,
  Grid2X2,
  Link,
  Loader2,
  Mic,
  MicOff,
  Play,
  Radio,
  Settings,
  ShieldCheck,
  Sparkles,
  Video,
} from 'lucide-react'
import { LiveKitRoom, VideoConference } from '@livekit/components-react'
import '@livekit/components-styles'
import './App.css'

const navItems = [
  { label: 'AlemLive', icon: Grid2X2, active: true },
]

function App() {
  const [form, setForm] = useState({
    roomName: import.meta.env.VITE_LIVEKIT_ROOM ?? 'alem-meeting',
    userName: import.meta.env.VITE_LIVEKIT_NAME ?? 'Мади Орысбек',
  })
  const [meeting, setMeeting] = useState(null)
  const [entryMode, setEntryMode] = useState('create')
  const [devices, setDevices] = useState({
    mic: true,
    camera: true,
  })
  const [isStarting, setIsStarting] = useState(false)
  const [joinError, setJoinError] = useState('')

  const canStart = form.userName.trim() && form.roomName.trim()
  const isConnected = Boolean(meeting)

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
    setDevices((current) => ({ ...current, [name]: !current[name] }))
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

  return (
    <main className="workspace-shell">
      <header className="topbar">
        <a className="brand" href="/">
          <span className="brand-mark">
            <Sparkles size={18} />
          </span>
          <span>Alem Workspace</span>
        </a>

        <nav className="workspace-nav" aria-label="Workspace navigation">
          {navItems.map(({ label, icon: Icon, active }) => (
            <button className={active ? 'nav-item active' : 'nav-item'} type="button" key={label}>
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
    </main>
  )
}

export default App
