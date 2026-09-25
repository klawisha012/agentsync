"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "../api";
import { dropAgent, handToken, listenerLabel, probeAgent } from "../agent";

export default function AccountView({ name }) {
  const [state, setState] = useState(null);
  const [explanation, setExplanation] = useState("");
  const [releaseID, setReleaseID] = useState(null);

  async function load() {
    const res = await api(`/accounts/${encodeURIComponent(name)}`);
    setState(res);
    return res;
  }

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

  async function confirmMachine() {
    setExplanation("");
    const probe = await probeAgent();
    if (!probe.ok) {
      setExplanation("Локальный агент не отвечает.");
      return;
    }
    if (!probe.id || !probe.host) {
      setExplanation("Локальный агент не назвал компьютер.");
      return;
    }
    const res = await api("/machine", {
      method: "POST",
      body: JSON.stringify({ id: probe.id, host: probe.host, listener: listenerLabel }),
    });
    if (!res.ok) {
      setExplanation(res.body?.explanation || "Не удалось подтвердить машину.");
      return;
    }
    const handed = await handToken(res.body.agentToken);
    await load();
    if (!handed) {
      setExplanation("Машина подтверждена на сайте. Локальный агент не принял вход.");
    }
  }

  async function release(id) {
    const res = await api(`/machine/${encodeURIComponent(id)}`, { method: "DELETE" });
    if (!res.ok) {
      setExplanation(res.body?.explanation || "Не удалось снять подтверждение.");
      return;
    }
    await dropAgent();
    setReleaseID(null);
    await load();
  }

  if (!state) {
    return <p className="lede">Открываем страницу…</p>;
  }
  if (!state.ok) {
    return <p className="explanation">{state.body?.explanation || "Страница не найдена."}</p>;
  }

  const page = state.body;
  const owner = Object.prototype.hasOwnProperty.call(page, "maskedMail");
  const machines = page.machines || [];

  return (
    <section className="profile">
      <div className="card-title">
        <h1 className="account-name">{page.name}</h1>
        {page.verified ? <em className="badge ok">верифицирован</em> : null}
      </div>
      {owner ? (
        <p className="hint">
          {page.verified ? "Почта подтверждена" : "Почта не подтверждена"}: <span className="mono">{page.maskedMail}</span>
          {page.letterPath ? (
            <>
              {" "}
              <Link href={page.letterPath}>Открыть письмо</Link>
            </>
          ) : null}
        </p>
      ) : null}
      <div className="stats">
        <div>
          <b>{page.views}</b>
          <span>просмотры</span>
        </div>
        <div>
          <b>{page.likes}</b>
          <span>лайки</span>
        </div>
      </div>
      {owner ? (
        <div className="machine">
          {machines.length === 0 ? (
            <div className="machine-row">
              <p>Машина не подтверждена.</p>
              <button className="solid" type="button" onClick={confirmMachine}>Подтвердить машину</button>
            </div>
          ) : (
            machines.map((item) => (
              <div className="machine-row" key={item.id}>
                <div>
                  <strong>{item.host}</strong>
                  <span className="live">Подтверждена</span>
                  <p className="hint">Идентификатор машины: {item.id}</p>
                  <p className="hint">Цепочка этого компьютера на месте.</p>
                </div>
                <button className="ghost" type="button" onClick={() => setReleaseID(item.id)}>
                  Снять подтверждение машины
                </button>
              </div>
            ))
          )}
        </div>
      ) : null}
      {explanation ? <p className="explanation">{explanation}</p> : null}
      {releaseID ? (
        <div className="dialog-back">
          <div className="dialog" role="dialog" aria-labelledby="unpair-title">
            <h2 id="unpair-title">Снять подтверждение машины?</h2>
            <p>Локальный агент выйдет из{"\u00a0"}аккаунта. Цепочка на диске останется.</p>
            <div className="dialog-actions">
              <button className="solid" type="button" onClick={() => release(releaseID)}>Снять подтверждение</button>
              <button className="ghost" type="button" onClick={() => setReleaseID(null)}>Отмена</button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
