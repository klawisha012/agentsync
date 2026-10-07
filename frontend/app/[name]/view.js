"use client";

import { useEffect, useState } from "react";
import { api } from "../api";
import Avatar from "../avatar";
import Icon from "../profile-icon";
import AgentBoard from "./agents";
import Comments from "./comments";

let accountInflight = null;

function fetchAccount(name) {
  if (accountInflight && accountInflight.name === name) {
    return accountInflight.promise;
  }
  const promise = api(`/accounts/${encodeURIComponent(name)}`).finally(() => {
    if (accountInflight && accountInflight.promise === promise) {
      accountInflight = null;
    }
  });
  accountInflight = { name, promise };
  return promise;
}

export default function AccountView({ name }) {
  const [state, setState] = useState(null);
  const [explanation, setExplanation] = useState("");
  const [releaseID, setReleaseID] = useState(null);

  async function load() {
    const res = await fetchAccount(name);
    setState(res);
    return res;
  }

  useEffect(() => {
    let gone = false;
    fetchAccount(name).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [name]);

  async function release(id) {
    const res = await api(`/machine/${encodeURIComponent(id)}`, { method: "DELETE" });
    if (!res.ok) {
      setExplanation(res.body?.explanation || "Не удалось снять подтверждение.");
      return;
    }
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
  const releasing = machines.find((item) => item.id === releaseID);

  return (
    <section className="profile">
      <div className="identity">
        <div className="identity-top">
          <div className="identity-main">
            <Avatar className="profile-mark" name={page.name} updated={page.avatarUpdated} letter={initials(page.name)} />
            <div className="identity-copy">
              <div className="identity-name">
                <h1>{page.name}</h1>
                {page.verified ? (
                  <em className="badge ok">
                    <Icon name="verified" />
                    верифицирован
                  </em>
                ) : null}
              </div>
              {owner ? null : <p className="hint">Публичная страница</p>}
            </div>
          </div>
          <div className="metric-row">
            <div className="metric">
              <span>Просмотры</span>
              <b><Icon name="eye" />{formatCount(page.views)}</b>
              <em>уникальные</em>
            </div>
            <div className="metric">
              <span>Лайки</span>
              <b><Icon name="heart" />{formatCount(page.likes)}</b>
              <em>по публикациям</em>
            </div>
          </div>
        </div>
        {owner && machines.length > 0 ? (
          <div className="machine-strip">
            {machines.map((item) => (
              <div className="machine-row" key={item.id}>
                <div className="machine-id">
                  <span className="machine-icon" aria-hidden="true"><Icon name="laptop" /></span>
                  <div>
                    <div className="machine-title">
                      <strong>{item.host}</strong>
                      <span className="live"><i />Подтверждена</span>
                    </div>
                    <p className="hint">
                      Идентификатор машины: <span className="mono">{item.id}</span>
                      {item.listener ? <> · <span className="mono">{item.listener}</span></> : null}
                    </p>
                  </div>
                </div>
                <div className="identity-actions">
                  <button className="warn" type="button" onClick={() => setReleaseID(item.id)}>
                    <Icon name="unlink" />
                    Снять подтверждение машины
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </div>
      <AgentBoard
        page={page}
        owner={owner}
        onChange={load}
        onExplain={setExplanation}
      />
      <Comments name={page.name} access={page.commentAccess || "hidden"} owner={owner} />
      {explanation ? <p className="explanation">{explanation}</p> : null}
      {releasing ? (
        <div className="dialog-back">
          <div className="dialog" role="dialog" aria-labelledby="unpair-title">
            <h2 id="unpair-title">Снять подтверждение машины?</h2>
            <p>
              Устройство <span className="mono">{releasing.host}</span> выйдет из{"\u00a0"}аккаунта. Снимки на диске останутся.
            </p>
            <div className="dialog-actions">
              <button className="ghost" type="button" onClick={() => setReleaseID(null)}>Отмена</button>
              <button className="warn solid-warn" type="button" onClick={() => release(releaseID)}>Снять подтверждение</button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}

function initials(name) {
  const parts = String(name).split("-").filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return String(name).slice(0, 2).toUpperCase();
}

function formatCount(value) {
  return String(value).replace(/\B(?=(\d{3})+(?!\d))/g, "\u202f");
}
