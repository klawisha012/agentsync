"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { api } from "../api";

export default function LoginCard() {
  const router = useRouter();
  const params = useSearchParams();
  const [tab, setTab] = useState(params.get("tab") === "register" ? "register" : "login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [show, setShow] = useState(false);
  const [explanation, setExplanation] = useState("");

  async function submit(event) {
    event.preventDefault();
    if (tab === "register") {
      if (!email.trim() || !password || !name.trim()) {
        setExplanation("Введите почту, пароль и\u00a0имя.");
        return;
      }
      const res = await api("/accounts", {
        method: "POST",
        body: JSON.stringify({ email, password, name }),
      });
      if (!res.ok) {
        setExplanation(res.body?.explanation || "Не удалось создать аккаунт.");
        return;
      }
      router.push(`/${res.body.name}`);
      router.refresh();
      return;
    }
    if (!email.trim() || !password) {
      setExplanation("Введите почту и\u00a0пароль.");
      return;
    }
    const res = await api("/session", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      setExplanation(res.body?.explanation || "Неверная почта или пароль.");
      return;
    }
    router.push(`/${res.body.name}`);
    router.refresh();
  }

  return (
    <div className="login-wrap">
      <form className="card" onSubmit={submit}>
        <div className="auth-mark" aria-hidden="true">AS</div>
        <div className="auth-title">
          <h1>AgentSync</h1>
        </div>
        <p className="lede">Безопасный перенос конфигураций AI-агентов</p>
        <div className="tabs" role="tablist">
          <button type="button" role="tab" aria-selected={tab === "login"} onClick={() => { setTab("login"); setExplanation(""); }}>
            Вход
          </button>
          <button type="button" role="tab" aria-selected={tab === "register"} onClick={() => { setTab("register"); setExplanation(""); }}>
            Регистрация
          </button>
        </div>
        {tab === "register" ? (
          <>
            <div className="row">
              <label htmlFor="name">Публичное имя страницы</label>
              <span className="hint">3–32 знака</span>
            </div>
            <input id="name" value={name} autoComplete="username" placeholder="alex-dev" onChange={(event) => setName(event.target.value)} />
            <p className="hint">Буквы любого алфавита, цифры и{"\u00a0"}дефис не по краям. Alice и{"\u00a0"}alice{"\u00a0"}— одно имя.</p>
          </>
        ) : null}
        <label htmlFor="email">Рабочая почта</label>
        <input id="email" type="email" autoComplete="email" value={email} placeholder="alex@domain.dev" onChange={(event) => setEmail(event.target.value)} />
        <div className="row">
          <label htmlFor="password">Пароль</label>
          {tab === "login" ? <a href="/recover">Восстановление пароля по почте</a> : null}
        </div>
        <div className="pass">
          <input
            id="password"
            type={show ? "text" : "password"}
            autoComplete={tab === "register" ? "new-password" : "current-password"}
            value={password}
            placeholder="••••••••"
            onChange={(event) => setPassword(event.target.value)}
          />
          <button type="button" onClick={() => setShow((value) => !value)}>
            {show ? "Скрыть" : "Показать"}
          </button>
        </div>
        {explanation ? <p className="explanation">{explanation}</p> : null}
        <button className="solid" type="submit">
          {tab === "register" ? "Зарегистрироваться" : "Войти в\u00a0аккаунт"}
        </button>
        <Link className="back-link" href="/">Вернуться на главную</Link>
      </form>
    </div>
  );
}
