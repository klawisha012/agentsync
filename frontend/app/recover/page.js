"use client";

import { useState } from "react";
import { api } from "../api";

export default function RecoverPage() {
  const [email, setEmail] = useState("");
  const [explanation, setExplanation] = useState("");

  async function submit(event) {
    event.preventDefault();
    if (!email.trim()) {
      setExplanation("Введите почту в\u00a0виде name@example.com.");
      return;
    }
    const res = await api("/recovery", {
      method: "POST",
      body: JSON.stringify({ email }),
    });
    setExplanation(res.body?.explanation || "Не удалось подготовить письмо.");
  }

  return (
    <section className="card">
      <h1>Новый пароль</h1>
      <p className="lede">Если эта почта есть в{"\u00a0"}аккаунте, ссылка для нового пароля придёт на неё, когда почтовый сервер будет подключён.</p>
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
    </section>
  );
}
