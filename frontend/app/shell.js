"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "./api";

const links = [
  { href: "/", label: "Главная" },
  { href: "/accounts", label: "Список аккаунтов" },
  { href: "/catalog", label: "Каталог" },
];

export default function Shell({ children }) {
  const path = usePathname();
  const router = useRouter();
  const [session, setSession] = useState(undefined);

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
          <span className="agent-pill">
            <i />
            локальный агент не отвечает
          </span>
          {session ? (
            <>
              <Link href={`/${session.name}`}>{session.name}</Link>
              <button className="ghost" type="button" onClick={logout}>
                Выйти
              </button>
            </>
          ) : (
            <>
              <Link className="ghost" href="/login">
                Войти
              </Link>
              <Link className="solid" href="/login?tab=register">
                Создать аккаунт
              </Link>
            </>
          )}
        </div>
      </header>
      <main className="main">{children}</main>
    </div>
  );
}
