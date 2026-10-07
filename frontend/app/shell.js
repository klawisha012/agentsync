"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { api } from "./api";
import Avatar from "./avatar";

const links = [
  { href: "/accounts", label: "Список аккаунтов" },
  { href: "/stats", label: "Статистика" },
  { href: "/docs", label: "Документация" },
];

export default function Shell({ children }) {
  const path = usePathname();
  const [session, setSession] = useState(undefined);
  const bare = path === "/login";

  useEffect(() => {
    if (path === "/login") {
      return undefined;
    }
    let gone = false;
    function load() {
      api("/session").then((res) => {
        if (!gone) {
          setSession(res.ok ? res.body : null);
        }
      });
    }
    load();
    window.addEventListener("agentsync-session", load);
    return () => {
      gone = true;
      window.removeEventListener("agentsync-session", load);
    };
  }, [path === "/login"]);

  if (bare) {
    return (
      <div className="shell bare">
        <main className="main">{children}</main>
      </div>
    );
  }

  return (
    <div className="shell">
      <header className="topbar">
        <div className="topbar-inner">
          <div className="brand-group">
            <Link className="brand" href="/">
              <span className="mark" aria-hidden="true">
                <svg viewBox="0 0 24 24" width="17" height="17">
                  <path fill="currentColor" d="M6.99 11 3 15l3.99 4v-3H14v-2H6.99v-3zM21 9l-3.99-4v3H10v2h7.01v3L21 9z" />
                </svg>
              </span>
              AgentSync
            </Link>
            <nav className="nav">
              {links.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  aria-current={path === item.href || (item.href === "/accounts" && path === "/catalog") ? "page" : undefined}
                >
                  {item.label}
                </Link>
              ))}
            </nav>
          </div>
          <div className="top-gap">
            {session ? (
              <ProfileMenu name={session.name} updated={session.avatarUpdated} />
            ) : session === null ? (
              <Link className="who" href="/login" aria-label="Войти">
                <span className="person" aria-hidden="true"><PersonIcon /></span>
              </Link>
            ) : null}
          </div>
        </div>
      </header>
      <main className="main">{children}</main>
    </div>
  );
}

function ProfileMenu({ name, updated }) {
  const router = useRouter();
  const box = useRef(null);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!open) {
      return undefined;
    }
    function onPointer(event) {
      if (box.current && !box.current.contains(event.target)) {
        setOpen(false);
      }
    }
    function onKey(event) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("pointerdown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  async function logout() {
    await api("/session", { method: "DELETE" });
    window.dispatchEvent(new Event("agentsync-session"));
    setOpen(false);
    router.push("/login");
  }

  return (
    <div className="profile-menu" ref={box}>
      <button
        className="who"
        type="button"
        aria-label="Профиль"
        aria-expanded={open}
        aria-haspopup="menu"
        onClick={() => setOpen((value) => !value)}
      >
        <span className="person" aria-hidden="true">
          {updated ? <Avatar className="person-photo" name={name} updated={updated} letter="" /> : <PersonIcon />}
        </span>
        <span className="who-name">{name}</span>
        <svg className="who-caret" viewBox="0 0 24 24" width="10" height="10" aria-hidden="true">
          <path fill="currentColor" d="M5 9l7 7 7-7z" />
        </svg>
      </button>
      {open ? (
        <div className="profile-pop" role="menu">
          <Link href={`/${name}`} role="menuitem" onClick={() => setOpen(false)}>Открыть мой профиль</Link>
          <Link href="/settings" role="menuitem" onClick={() => setOpen(false)}>Настройки</Link>
          <div className="menu-sep" role="separator" />
          <button className="menu-leave" type="button" role="menuitem" onClick={logout}>Выйти из аккаунта…</button>
        </div>
      ) : null}
    </div>
  );
}

function PersonIcon() {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16">
      <path fill="currentColor" d="M12 12a4 4 0 1 0-4-4 4 4 0 0 0 4 4zm0 2c-3.3 0-8 1.7-8 4v1h16v-1c0-2.3-4.7-4-8-4z" />
    </svg>
  );
}
