"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { api } from "../../api";

export default function NewPasswordPage() {
  const params = useParams();
  const [password, setPassword] = useState("");
  const [explanation, setExplanation] = useState("");
  const [done, setDone] = useState(false);

  async function submit(event) {
    event.preventDefault();
    if (!password) {
      setExplanation("Введите новый пароль.");
      return;
    }
    const res = await api("/recovery/password", {
      method: "POST",
      body: JSON.stringify({ token: params.token, password }),
    });
    if (!res.ok) {
      setExplanation(res.body?.explanation || "Ссылка из письма недействительна или устарела.");
      return;
    }
    setDone(true);
    setExplanation("");
  }

  return (
    <section className="card">
      <h1>Новый пароль</h1>
      {done ? (
        <>
          <p className="lede">Пароль изменён. Войдите с{"\u00a0"}новым паролем в{"\u00a0"}тот же аккаунт.</p>
          <Link className="solid" href="/login">Войти</Link>
        </>
      ) : (
        <form onSubmit={submit}>
          <label htmlFor="new-password">Новый пароль</label>
          <input
            id="new-password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <button className="solid" type="submit">Задать пароль</button>
          {explanation ? <p className="explanation">{explanation}</p> : null}
        </form>
      )}
    </section>
  );
}
