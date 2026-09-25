"use client";

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
        setExplanation("Введите почту, пароль и имя.");
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
      setExplanation("Введите почту и пароль.");
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
        <h1>AgentSync</h1>
        <p className="lede">Вход в страницу аккаунта</p>
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
            <label htmlFor="name">Публичное имя</label>
            <input id="name" value={name} autoComplete="username" onChange={(event) => setName(event.target.value)} />
            <p className="hint">3–32 знака: буквы любого алфавита, цифры и дефис не по краям. Alice и alice — одно имя.</p>
          </>
        ) : null}
        <label htmlFor="email">Рабочая почта</label>
        <input id="email" type="email" autoComplete="email" value={email} placeholder="name@example.com" onChange={(event) => setEmail(event.target.value)} />
        <div className="row">
          <label htmlFor="password">Пароль</label>
          <a href="/recover">Восстановление пароля по почте</a>
        </div>
        <div className="pass">
          <input
            id="password"
            type={show ? "text" : "password"}
            autoComplete={tab === "register" ? "new-password" : "current-password"}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <button type="button" onClick={() => setShow((value) => !value)}>
            {show ? "Скрыть" : "Показать"}
          </button>
        </div>
        {explanation ? <p className="explanation">{explanation}</p> : null}
        <button className="solid" type="submit">
          {tab === "register" ? "Создать аккаунт" : "Войти в аккаунт"}
        </button>
      </form>
    </div>
  );
}
