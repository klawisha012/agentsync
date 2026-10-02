"use client";

import { useEffect, useState } from "react";
import { api } from "../api";
import Avatar from "../avatar";
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
        <span className="profile-glow" aria-hidden="true" />
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

function Icon({ name }) {
  const paths = {
    edit: "M4 17.5V20h2.5L17.8 8.7l-2.5-2.5L4 17.5zm14.7-9.2a1 1 0 0 0 0-1.4l-1.6-1.6a1 1 0 0 0-1.4 0l-1.1 1.1 3 3 1.1-1.1z",
    verified: "M9.2 16.2 5.5 12.5l1.4-1.4 2.3 2.3 6-6 1.4 1.4-7.4 7.4z",
    mail: "M4 6h16v12H4V6zm8 6.2L18.2 8H5.8L12 12.2z",
    eye: "M12 6c4.5 0 8.2 2.8 9.5 6-1.3 3.2-5 6-9.5 6S3.8 15.2 2.5 12C3.8 8.8 7.5 6 12 6zm0 2a4 4 0 1 0 0 8 4 4 0 0 0 0-8z",
    heart: "M12 19s-6.5-4.1-8.2-8.1C2.6 8.4 4 6 6.6 6c1.6 0 2.6.8 3.4 1.8C10.8 6.8 11.8 6 13.4 6 16 6 17.4 8.4 16.2 10.9 14.5 14.9 12 19 12 19z",
    laptop: "M4 6h16v10H4V6zm-1 12h18v1.5H3V18z",
    unlink: "M8.5 13.5 6 16a3 3 0 0 0 4.2 4.2l2.5-2.5-1.4-1.4-2.5 2.5a1 1 0 0 1-1.4-1.4l2.5-2.5-1.4-1.4zm7-3 2.5-2.5A3 3 0 0 0 13.8 3.8L11.3 6.3l1.4 1.4 2.5-2.5a1 1 0 0 1 1.4 1.4L14.1 9.1l1.4 1.4z",
    delete: "M8 4h8l1 2h4v2H3V6h4l1-2zm1 6h2v8H9v-8zm4 0h2v8h-2v-8z",
  };
  return (
    <svg className="ico" viewBox="0 0 24 24" width="16" height="16" aria-hidden="true">
      <path fill="currentColor" d={paths[name]} />
    </svg>
  );
}
