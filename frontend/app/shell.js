"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "./api";
import { listenerLabel, useAgentProbe } from "./agent";

const links = [
  { href: "/", label: "Главная" },
  { href: "/accounts", label: "Список аккаунтов" },
  { href: "/catalog", label: "Каталог" },
];

export default function Shell({ children }) {
  const path = usePathname();
  const router = useRouter();
  const [session, setSession] = useState(undefined);
  const [agent] = useAgentProbe();
  const sameAccount = Boolean(
    agent.ok &&
      session &&
      agent.account &&
      agent.account.localeCompare(session.name, "ru", { sensitivity: "accent" }) === 0,
  );
  const otherAccount = Boolean(agent.ok && session && agent.account && !sameAccount);
  let agentLabel = "локальный агент не отвечает";
  if (agent.ok) {
    const host = agent.host ? ` · ${agent.host}` : "";
    const state = sameAccount ? " · онлайн" : otherAccount ? " · другой аккаунт" : "";
    agentLabel = `${listenerLabel}${host} · ${agent.ms}\u00a0мс${state}`;
  }

  useEffect(() => {
    let gone = false;
    api("/session").then((res) => {
      if (!gone) {
        setSession(res.ok ? res.body : null);
      }
    });
    return () => {
      gone = true;
    };
  }, [path]);

  async function logout() {
    await api("/session", { method: "DELETE" });
    setSession(null);
    router.push("/login");
    router.refresh();
  }

  return (
    <div className="shell">
      <header className="topbar">
        <Link className="brand" href="/">
          AgentSync
        </Link>
        <nav className="nav">
          {links.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              aria-current={path === item.href ? "page" : undefined}
            >
              {item.label}
            </Link>
          ))}
        </nav>
        <div className="top-gap">
          <span className={sameAccount ? "agent-pill online" : "agent-pill"}>
            <i />
            {agentLabel}
          </span>
          {session ? (
            <Link className="who" href={`/${session.name}`}>
              {session.name}
            </Link>
          ) : (
            <Link className="ghost" href="/login">
              Войти
            </Link>
          )}
          <Link className="solid" href="/login?tab=register">
            Создать аккаунт
          </Link>
          {session ? (
            <button className="ghost" type="button" onClick={logout}>
              Выйти
            </button>
          ) : null}
        </div>
      </header>
      <main className="main">{children}</main>
    </div>
  );
}
