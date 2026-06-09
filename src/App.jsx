import { useMemo, useState } from 'react'
import {
  Bell,
  Bot,
  CalendarDays,
  ChevronDown,
  Contact,
  Copy,
  Grid2X2,
  Link,
  Mic,
  MonitorUp,
  NotebookTabs,
  Play,
  Radio,
  Send,
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

const agenda = [
  { time: '09:30', title: 'Daily sync', tone: 'blue' },
  { time: '12:00', title: 'Product review', tone: 'green' },
  { time: '16:15', title: 'LiveKit test call', tone: 'violet' },
]

function App() {
  const [form, setForm] = useState({
    serverUrl: import.meta.env.VITE_LIVEKIT_URL ?? '',
    token: import.meta.env.VITE_LIVEKIT_TOKEN ?? '',
    roomName: import.meta.env.VITE_LIVEKIT_ROOM ?? 'alem-meeting',
    userName: import.meta.env.VITE_LIVEKIT_NAME ?? 'Мади Орысбек',
  })
  const [meeting, setMeeting] = useState(null)

  const canJoin = form.serverUrl.trim() && form.token.trim()
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
  }

  function joinMeeting(event) {
    event.preventDefault()

    if (!canJoin) {
      return
    }

    setMeeting({
      serverUrl: form.serverUrl.trim(),
      token: form.token.trim(),
      roomName: form.roomName.trim(),
      userName: form.userName.trim(),
    })
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
          <span className="date-label">Tue, June 9</span>
          <h1 id="meeting-title">Meeting room для команды</h1>
          <p>Подключите LiveKit, проверьте камеру и начните созвон без выхода из рабочего пространства.</p>
        </div>

        <div className="hero-actions">
          <button className="primary-action" type="button">
            <Sparkles size={20} />
            Спросить AI
          </button>
          <button className="soft-action" type="button">
            <CalendarDays size={19} />
            План созвона
          </button>
          <button className="soft-action" type="button">
            <Send size={18} />
            Пригласить
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
                <p>{isConnected ? 'Комната активна' : 'Введите URL и token для подключения'}</p>
              </div>
            </div>

            <form className="join-form" onSubmit={joinMeeting}>
              <label>
                <span>LiveKit URL</span>
                <input
                  name="serverUrl"
                  value={form.serverUrl}
                  onChange={updateField}
                  placeholder="wss://your-project.livekit.cloud"
                  autoComplete="url"
                />
              </label>

              <label>
                <span>Access token</span>
                <textarea
                  name="token"
                  value={form.token}
                  onChange={updateField}
                  placeholder="eyJhbGciOiJIUzI1NiIs..."
                  rows="4"
                />
              </label>

              <div className="form-row">
                <label>
                  <span>Room</span>
                  <input name="roomName" value={form.roomName} onChange={updateField} />
                </label>
                <label>
                  <span>Name</span>
                  <input name="userName" value={form.userName} onChange={updateField} />
                </label>
              </div>

              <button className="join-button" type="submit" disabled={!canJoin}>
                <Play size={18} fill="currentColor" />
                {isConnected ? 'Переподключиться' : 'Войти в комнату'}
              </button>
            </form>
          </section>

          <section className="panel compact-panel">
            <div className="panel-heading">
              <span className="panel-icon">
                <ShieldCheck size={21} />
              </span>
              <div>
                <h2>Перед эфиром</h2>
                <p>Быстрая проверка устройства</p>
              </div>
            </div>

            <div className="check-list">
              <div>
                <Mic size={18} />
                <span>Микрофон готов</span>
              </div>
              <div>
                <Video size={18} />
                <span>Камера подключена</span>
              </div>
              <div>
                <MonitorUp size={18} />
                <span>Screen share доступен</span>
              </div>
            </div>
          </section>

          <section className="panel agenda-panel">
            <div className="panel-heading">
              <span className="panel-icon">
                <CalendarDays size={21} />
              </span>
              <div>
                <h2>Календарь</h2>
                <p>Сегодня</p>
              </div>
            </div>

            <div className="agenda-list">
              {agenda.map((item) => (
                <div className={`agenda-item ${item.tone}`} key={item.title}>
                  <time>{item.time}</time>
                  <span>{item.title}</span>
                </div>
              ))}
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
                audio
                video
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
                <h3>Готово к подключению</h3>
                <p>Укажите LiveKit URL и access token слева. Token должен быть создан на backend для этой комнаты и пользователя.</p>
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

          <section className="panel notes-panel">
            <div className="panel-heading">
              <span className="panel-icon">
                <NotebookTabs size={21} />
              </span>
              <div>
                <h2>Заметки</h2>
                <p>AI summary после звонка</p>
              </div>
            </div>
            <textarea placeholder="Ключевые решения, action items, ссылки..." rows="8" />
          </section>
        </aside>
      </section>
    </main>
  )
}

export default App
