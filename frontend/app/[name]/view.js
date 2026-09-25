"use client";

import { useEffect, useState } from "react";
import { api } from "../api";

export default function AccountView({ name }) {
  const [state, setState] = useState(null);

  useEffect(() => {
    let gone = false;
    api(`/accounts/${encodeURIComponent(name)}`).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [name]);

  if (!state) {
    return <p className="lede">Открываем страницу…</p>;
  }
  if (!state.ok) {
    return <p className="explanation">{state.body?.explanation || "Страница не найдена."}</p>;
  }

  return (
    <section>
      <h1 className="account-name">{state.body.name}</h1>
      <div className="stats">
        <div>
          <b>{state.body.views}</b>
          <span>просмотры</span>
        </div>
        <div>
          <b>{state.body.likes}</b>
          <span>лайки</span>
        </div>
      </div>
    </section>
  );
}
