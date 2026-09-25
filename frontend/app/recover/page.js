"use client";

import Link from "next/link";
import { useState } from "react";
import { api } from "../api";

export default function RecoverPage() {
  const [email, setEmail] = useState("");
  const [explanation, setExplanation] = useState("");
  const [letterPath, setLetterPath] = useState("");

  async function submit(event) {
    event.preventDefault();
    setLetterPath("");
    if (!email.trim()) {
      setExplanation("Введите почту в\u00a0виде name@example.com.");
      return;
    }
    const res = await api("/recovery", {
      method: "POST",
      body: JSON.stringify({ email }),
    });
    setExplanation(res.body?.explanation || "Не удалось подготовить письмо.");
    if (res.ok && res.body?.letterPath) {
      setLetterPath(res.body.letterPath);
    }
  }

  return (
    <section className="card">
      <h1>Новый пароль</h1>
      <p className="lede">Письмо со{"\u00a0"}ссылкой приходит на почту аккаунта. Пока почтовый сервер не подключён, ссылка открывается здесь.</p>
      <form onSubmit={submit}>
        <label htmlFor="recover-email">Рабочая почта</label>
        <input
          id="recover-email"
          type="email"
          autoComplete="email"
          placeholder="name@example.com"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
        <button className="solid" type="submit">Прислать ссылку</button>
      </form>
      {explanation ? <p className="hint">{explanation}</p> : null}
      {letterPath ? (
        <p className="hint">
          <Link href={letterPath}>Открыть письмо и{"\u00a0"}задать новый пароль</Link>
        </p>
      ) : null}
    </section>
  );
}
