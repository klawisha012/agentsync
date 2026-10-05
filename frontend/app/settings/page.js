"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../api";
import Avatar from "../avatar";

const parts = [
  { id: "account", label: "Аккаунт" },
  { id: "privacy", label: "Приватность" },
];

export default function SettingsPage() {
  const router = useRouter();
  const [session, setSession] = useState(undefined);
  const [page, setPage] = useState(null);
  const [part, setPart] = useState("account");
  const [note, setNote] = useState(null);
  const [nickname, setNickname] = useState("");
  const [email, setEmail] = useState("");
  const [emailPassword, setEmailPassword] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [nextPassword, setNextPassword] = useState("");
  const [repeatPassword, setRepeatPassword] = useState("");
  const [shown, setShown] = useState({});
  const [deletePassword, setDeletePassword] = useState("");
  const [share, setShare] = useState({ copy: false, view: false, versions: false, comments: "hidden" });
  const [avatarFile, setAvatarFile] = useState(null);
  const [avatarPreview, setAvatarPreview] = useState("");

  useEffect(() => {
    let gone = false;
    api("/session").then((res) => {
      if (gone) {
        return null;
      }
      if (!res.ok) {
        router.replace("/login");
        return null;
      }
      setSession(res.body);
      setNickname(res.body.name);
      return api(`/accounts/${encodeURIComponent(res.body.name)}`);
    }).then((res) => {
      if (!res || gone || !res.ok) {
        return;
      }
      setPage(res.body);
      const comments = ["open", "users", "hidden"].includes(res.body.commentAccess) ? res.body.commentAccess : "hidden";
      setShare({
        copy: Boolean(res.body.shareCopy),
        view: Boolean(res.body.shareView),
        versions: Boolean(res.body.shareVersions),
        comments,
      });
    });
    return () => {
      gone = true;
    };
  }, [router]);

  function tell(text, bad) {
    setNote({ text, bad });
  }

  async function saveName(event) {
    event.preventDefault();
    const res = await api("/account/name", { method: "POST", body: JSON.stringify({ name: nickname }) });
    if (!res.ok) {
      tell(res.body?.explanation || "Не удалось сменить имя.", true);
      return;
    }
    setSession((current) => ({ ...current, name: res.body.name }));
    window.dispatchEvent(new Event("agentsync-session"));
    tell("Имя сохранено.", false);
  }

  async function saveEmail(event) {
    event.preventDefault();
    const res = await api("/account/email", {
      method: "POST",
      body: JSON.stringify({ email, password: emailPassword }),
    });
    if (!res.ok) {
      tell(res.body?.explanation || "Не удалось сменить почту.", true);
      return;
    }
    setPage((current) => ({
      ...(current || {}),
      maskedMail: res.body.maskedMail,
      verified: false,
      letterPath: res.body.letterPath,
    }));
    setEmail("");
    setEmailPassword("");
    tell("Почта изменена. Подтвердите её по ссылке из письма.", false);
  }

  async function savePassword(event) {
    event.preventDefault();
    if (nextPassword !== repeatPassword) {
      tell("Пароли не совпадают.", true);
      return;
    }
    const res = await api("/account/password", {
      method: "POST",
      body: JSON.stringify({ current: currentPassword, next: nextPassword }),
    });
    if (!res.ok) {
      tell(res.body?.explanation || "Не удалось сменить пароль.", true);
      return;
    }
    setCurrentPassword("");
    setNextPassword("");
    setRepeatPassword("");
    tell("Пароль изменён.", false);
  }

  async function savePrivacy(event) {
    event.preventDefault();
    const res = await api("/account/privacy", { method: "POST", body: JSON.stringify(share) });
    if (!res.ok) {
      tell(res.body?.explanation || "Не удалось сохранить приватность.", true);
      return;
    }
    tell("Приватность сохранена.", false);
  }

  function chooseAvatar(event) {
    const file = event.target.files?.[0] || null;
    setAvatarFile(file);
    setAvatarPreview((current) => {
      if (current) {
        URL.revokeObjectURL(current);
      }
      return file ? URL.createObjectURL(file) : "";
    });
  }

  async function saveAvatar(event) {
    event.preventDefault();
    if (!avatarFile) {
      tell("Выберите файл.", true);
      return;
    }
    const data = new FormData();
    data.append("file", avatarFile);
    const res = await fetch("/api/account/avatar", { method: "POST", body: data, credentials: "same-origin" });
    const body = await res.json().catch(() => null);
    if (!res.ok) {
      tell(body?.explanation || "Не удалось сохранить аватар.", true);
      return;
    }
    if (avatarPreview) {
      URL.revokeObjectURL(avatarPreview);
    }
    setAvatarPreview("");
    setAvatarFile(null);
    setPage((current) => ({ ...(current || {}), hasAvatar: true, avatarUpdated: body.avatarUpdated }));
    window.dispatchEvent(new Event("agentsync-session"));
    tell("Аватар сохранён.", false);
  }

  async function removeAvatar() {
    const res = await fetch("/api/account/avatar", { method: "DELETE", credentials: "same-origin" });
    const body = await res.json().catch(() => null);
    if (!res.ok) {
      tell(body?.explanation || "Не удалось удалить аватар.", true);
      return;
    }
    setPage((current) => ({ ...(current || {}), hasAvatar: false, avatarUpdated: null }));
    window.dispatchEvent(new Event("agentsync-session"));
    tell("Аватар удалён.", false);
  }

  async function remove(event) {
    event.preventDefault();
    const res = await api("/account/delete", {
      method: "POST",
      body: JSON.stringify({ password: deletePassword }),
    });
    if (!res.ok) {
      tell(res.body?.explanation || "Не удалось удалить аккаунт.", true);
      return;
    }
    await api("/session", { method: "DELETE" });
    window.dispatchEvent(new Event("agentsync-session"));
    router.push("/login");
  }

  if (!session) {
    return <p className="lede">Открываем настройки…</p>;
  }

  return (
    <section className="settings-layout">
      <aside className="settings-side" aria-label="Разделы настроек">
        <p className="settings-who">
          {page?.avatarUpdated ? <Avatar className="settings-avatar" name={session.name} updated={page.avatarUpdated} letter="" /> : null}
          {session.name}
        </p>
        {parts.map((item) => (
          <button
            key={item.id}
            type="button"
            aria-current={part === item.id ? "page" : undefined}
            onClick={() => { setPart(item.id); setNote(null); }}
          >
            {item.label}
          </button>
        ))}
      </aside>
      <div className="settings-main">
        <h1>{parts.find((item) => item.id === part)?.label}</h1>
        {note ? <p className={note.bad ? "explanation" : "hint"}>{note.text}</p> : null}
        {part === "account" ? (
          <>
            <section className="settings-block">
              <h2>Аватар</h2>
              <form className="settings-form" onSubmit={saveAvatar}>
                <div className="avatar-row">
                  {avatarPreview ? (
                    <img className="settings-avatar lg" src={avatarPreview} alt="" />
                  ) : (
                    <Avatar className="settings-avatar lg" name={session.name} updated={page?.avatarUpdated} letter={session.name.slice(0, 2).toUpperCase()} />
                  )}
                  <div className="avatar-fields">
                    <label htmlFor="settings-avatar">Файл</label>
                    <input id="settings-avatar" type="file" accept="image/png,image/jpeg,image/gif,.png,.jpg,.jpeg,.gif" onChange={chooseAvatar} />
                    <p className="hint">{"PNG, JPEG или GIF, до\u00a02\u00a0МБ. Сторона до\u00a01\u202f024 пикселей. GIF остаётся анимацией."}</p>
                    <div className="settings-row">
                      <button className="solid" type="submit">Сохранить</button>
                      {page?.hasAvatar ? <button className="danger" type="button" onClick={removeAvatar}>Удалить аватар</button> : null}
                    </div>
                  </div>
                </div>
              </form>
            </section>
            <section className="settings-block">
              <h2>Почта</h2>
              <p className="mail-line">
                {page?.verified ? "Почта подтверждена" : "Почта не подтверждена"}
                {page?.maskedMail ? <>: <span className="mono">{page.maskedMail}</span></> : null}
                {page?.letterPath ? <> · <Link href={page.letterPath}>Открыть письмо</Link></> : null}
              </p>
              <form className="settings-form" onSubmit={saveEmail}>
                <label htmlFor="settings-email">Новая почта</label>
                <input id="settings-email" type="email" value={email} autoComplete="email" onChange={(event) => setEmail(event.target.value)} />
                <label htmlFor="settings-email-password">Текущий пароль</label>
                <div className="settings-row">
                  <SecretField id="settings-email-password" shown={shown.email} onToggle={() => toggleShown("email")} value={emailPassword} autoComplete="current-password" onChange={(event) => setEmailPassword(event.target.value)} />
                  <button className="solid" type="submit">Сохранить</button>
                </div>
              </form>
            </section>
            <section className="settings-block">
              <h2>Имя</h2>
              <form className="settings-form" onSubmit={saveName}>
                <label htmlFor="settings-name">Имя</label>
                <div className="settings-row">
                  <input id="settings-name" value={nickname} maxLength={32} onChange={(event) => setNickname(event.target.value)} />
                  <button className="solid" type="submit">Сохранить</button>
                </div>
                <p className="hint">{"От 3 до 32 знаков: буквы, цифры и\u00a0дефис не по краям."}</p>
              </form>
            </section>
            <section className="settings-block">
              <h2>Пароль</h2>
              <form className="settings-form" onSubmit={savePassword}>
                <label htmlFor="settings-current">Текущий пароль</label>
                <SecretField id="settings-current" shown={shown.current} onToggle={() => toggleShown("current")} value={currentPassword} autoComplete="current-password" onChange={(event) => setCurrentPassword(event.target.value)} />
                <label htmlFor="settings-next">Новый пароль</label>
                <SecretField id="settings-next" shown={shown.next} onToggle={() => toggleShown("next")} value={nextPassword} autoComplete="new-password" onChange={(event) => setNextPassword(event.target.value)} />
                <label htmlFor="settings-repeat">Повторите пароль</label>
                <div className="settings-row">
                  <SecretField id="settings-repeat" shown={shown.repeat} onToggle={() => toggleShown("repeat")} value={repeatPassword} autoComplete="new-password" onChange={(event) => setRepeatPassword(event.target.value)} />
                  <button className="solid" type="submit">Сохранить</button>
                </div>
              </form>
            </section>
            <section className="settings-block">
              <h2>Удаление</h2>
              <form className="settings-form" onSubmit={remove}>
                <p className="hint">Публикации будут сняты, имя освободится. Файлы на машине останутся.</p>
                <label htmlFor="settings-delete">Пароль</label>
                <div className="settings-row">
                  <SecretField id="settings-delete" shown={shown.remove} onToggle={() => toggleShown("remove")} value={deletePassword} autoComplete="current-password" onChange={(event) => setDeletePassword(event.target.value)} />
                  <button className="danger solid-danger" type="submit">Удалить</button>
                </div>
              </form>
            </section>
          </>
        ) : null}
        {part === "privacy" ? <PrivacyForm share={share} setShare={setShare} onSubmit={savePrivacy} /> : null}
      </div>
    </section>
  );

  function toggleShown(key) {
    setShown((current) => ({ ...current, [key]: !current[key] }));
  }
}

const summaryWord = { open: "Открытый", hidden: "Скрытый", partial: "Частичный" };

const profileCopy = {
  open: "В\u00a0открытом профиле другие копируют команду применения, открывают текущую версию и\u00a0все версии, оставляют комментарии.",
  hidden: "В\u00a0скрытом профиле эти действия видны только вам.",
  partial: "Часть действий открыта, часть скрыта. «Открытый» или «Скрытый» ставит все пункты одинаково.",
};

const commentHelp = {
  open: "Комментарии видны всем. Оставить свой можно после входа.",
  users: "Комментарии видят только вошедшие. Без входа раздел скрыт.",
  hidden: "Раздел скрыт. Комментарии видите и\u00a0пишете только вы.",
};

function profileMode(share) {
  const opened = share.copy && share.view && share.versions && share.comments === "open";
  const closed = !share.copy && !share.view && !share.versions && share.comments === "hidden";
  if (opened) {
    return "open";
  }
  if (closed) {
    return "hidden";
  }
  return "partial";
}

function PrivacyForm({ share, setShare, onSubmit }) {
  const mode = profileMode(share);

  function applyProfile(value) {
    if (value === "open") {
      setShare({ copy: true, view: true, versions: true, comments: "open" });
    }
    if (value === "hidden") {
      setShare({ copy: false, view: false, versions: false, comments: "hidden" });
    }
  }

  return (
    <form className="steam" onSubmit={onSubmit}>
      <p className="steam-summary">
        <span className="steam-label">{"Доступ к\u00a0основным данным:"}</span>
        <span className="steam-value">{summaryWord[mode]}</span>
      </p>
      <p className="steam-help">
        {"Имя страницы видно всем. Команда применения, осмотр версий и\u00a0комментарии настраиваются ниже. Снятие публикации остаётся только у\u00a0вас."}
      </p>
      <div className="steam-block">
        <div className="steam-row">
          <label className="steam-label" htmlFor="privacy-profile">Мой профиль:</label>
          <select id="privacy-profile" value={mode} onChange={(event) => applyProfile(event.target.value)}>
            {mode === "partial" ? <option value="partial">Частичный</option> : null}
            <option value="open">Открытый</option>
            <option value="hidden">Скрытый</option>
          </select>
        </div>
        <p className="steam-help">{profileCopy[mode]}</p>
      </div>
      <p className="steam-lead">Следующие элементы можно настроить отдельно:</p>
      <SteamFlag
        id="privacy-copy"
        label="Команда применения:"
        value={share.copy}
        onChange={(copy) => setShare({ ...share, copy })}
        help={"Другие копируют команду agentsync apply с\u00a0вашим именем и\u00a0агентом"}
      />
      <SteamFlag
        id="privacy-view"
        label="Осмотр текущей версии:"
        value={share.view}
        onChange={(view) => setShare({ ...share, view })}
        help="Другие открывают манифест последней опубликованной версии"
      />
      <SteamFlag
        id="privacy-versions"
        label="Все версии:"
        value={share.versions}
        onChange={(versions) => setShare({ ...share, versions })}
        help={"Другие открывают список версий и\u00a0каждую из\u00a0них"}
      />
      <div className="steam-block">
        <div className="steam-row">
          <label className="steam-label" htmlFor="privacy-comments">Раздел комментариев:</label>
          <select id="privacy-comments" className="steam-wide" value={share.comments} onChange={(event) => setShare({ ...share, comments: event.target.value })}>
            <option value="open">Оставлять комментарии могут все</option>
            <option value="users">Оставлять комментарии могут только вошедшие</option>
            <option value="hidden">Комментарии скрыты</option>
          </select>
        </div>
        <p className="steam-help">{commentHelp[share.comments] || commentHelp.hidden}</p>
      </div>
      <button className="solid steam-save" type="submit">Сохранить</button>
    </form>
  );
}

function SteamFlag({ id, label, value, onChange, help }) {
  return (
    <div className="steam-block">
      <div className="steam-row">
        <label className="steam-label" htmlFor={id}>{label}</label>
        <select id={id} value={value ? "open" : "hidden"} onChange={(event) => onChange(event.target.value === "open")}>
          <option value="open">Открытый</option>
          <option value="hidden">Скрытый</option>
        </select>
      </div>
      <p className="steam-help">{help}</p>
    </div>
  );
}

function SecretField({ id, value, onChange, autoComplete, shown, onToggle }) {
  return (
    <div className="pass">
      <input id={id} type={shown ? "text" : "password"} value={value} autoComplete={autoComplete} onChange={onChange} />
      <button type="button" aria-label={shown ? "Скрыть пароль" : "Показать пароль"} onClick={onToggle}>
        <EyeIcon open={shown} />
      </button>
    </div>
  );
}

function EyeIcon({ open }) {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
      <path fill="currentColor" d="M12 6c4.5 0 8.2 2.8 9.5 6-1.3 3.2-5 6-9.5 6S3.8 15.2 2.5 12C3.8 8.8 7.5 6 12 6zm0 2a4 4 0 1 0 0 8 4 4 0 0 0 0-8z" />
      {open ? <path fill="currentColor" d="M4.2 3.3 5.6 1.9 22.1 18.4 20.7 19.8z" /> : null}
    </svg>
  );
}
