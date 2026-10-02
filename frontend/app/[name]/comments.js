"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api } from "../api";
import Avatar from "../avatar";

const ownerHint = {
  hidden: "Другие этот раздел не видят",
  users: "Комментарии видят только те, кто вошёл",
  open: "Комментарии видят все",
};

const months = ["янв.", "фев.", "мар.", "апр.", "мая", "июн.", "июл.", "авг.", "сент.", "окт.", "нояб.", "дек."];
const commentLimit = 1000;

export default function Comments({ name, access, owner }) {
  const [state, setState] = useState(null);
  const [text, setText] = useState("");
  const [note, setNote] = useState("");
  const [pending, setPending] = useState(null);
  const [page, setPage] = useState(1);

  function load(nextPage = page) {
    return api(`/accounts/${encodeURIComponent(name)}/comments?page=${nextPage}`).then((res) => {
      setState(res);
      return res;
    });
  }

  useEffect(() => {
    setPage(1);
  }, [name, access]);

  useEffect(() => {
    let gone = false;
    api(`/accounts/${encodeURIComponent(name)}/comments?page=${page}`).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [name, access, page]);

  if (access === "hidden" && !owner) {
    return null;
  }
  if (!state) {
    return (
      <section className="comments">
        <div className="comments-bar"><h2>Комментарии</h2></div>
        <p className="hint">Открываем комментарии…</p>
      </section>
    );
  }
  if (!state.ok) {
    return (
      <section className="comments">
        <div className="comments-bar"><h2>Комментарии</h2></div>
        <p className="explanation">{state.body?.explanation || "Не удалось открыть комментарии."}</p>
      </section>
    );
  }

  async function send(event) {
    event.preventDefault();
    const res = await api(`/accounts/${encodeURIComponent(name)}/comments`, {
      method: "POST",
      body: JSON.stringify({ body: text }),
    });
    if (!res.ok) {
      setNote(res.body?.explanation || "Не удалось отправить комментарий.");
      return;
    }
    setText("");
    setNote("");
    if (page === 1) {
      await load(1);
      return;
    }
    setPage(1);
  }

  async function remove(id) {
    const res = await api(`/accounts/${encodeURIComponent(name)}/comments/${encodeURIComponent(id)}`, { method: "DELETE" });
    if (!res.ok) {
      setNote(res.body?.explanation || "Не удалось удалить комментарий.");
      return;
    }
    setPending(null);
    setNote("");
    const next = await load(page);
    if (next.ok && next.body.page && next.body.page !== page) {
      setPage(next.body.page);
    }
  }

  const hidden = Boolean(state.body.hidden);
  const comments = state.body.comments || [];
  const pages = state.body.pages || 0;
  const current = state.body.page || page;

  return (
    <section className="comments">
      <div className="comments-bar">
        <h2>Комментарии</h2>
      </div>
      {owner && ownerHint[access] ? <p className="hint">{ownerHint[access]}</p> : null}
      {hidden && !owner ? (
        <p className="hint">
          <Link href="/login">Войдите</Link>, чтобы видеть комментарии
        </p>
      ) : null}
      {state.body.canPost ? (
        <form className="comment-form" onSubmit={send}>
          <label htmlFor="comment-body">Комментарий</label>
          <textarea id="comment-body" value={text} rows={3} aria-describedby="comment-limit" onChange={(event) => setText(clipComment(event.target.value))} />
          <div className="comment-submit">
            <button className="solid" type="submit">Отправить</button>
            <p id="comment-limit" className={commentLength(text) >= commentLimit ? "comment-count full" : "comment-count"}>{`${commentLength(text)}\u00a0из\u00a01\u202f000`}</p>
          </div>
        </form>
      ) : null}
      {!hidden && !state.body.canPost ? (
        <p className="hint">
          <Link href="/login">Войдите</Link>, чтобы оставить комментарий
        </p>
      ) : null}
      {!hidden ? (
        comments.length === 0 ? <p className="hint">Комментариев пока нет</p> : (
          <ul className="comment-list">
            {comments.map((item) => (
              <li key={item.id}>
                <Avatar className="comment-avatar" name={item.author} updated={item.avatarUpdated} letter={mark(item.author)} />
                <div className="comment-body">
                  <Link href={`/${encodeURIComponent(item.author)}`}>{item.author}</Link>
                  <time dateTime={item.created}>{stamp(item.created)}</time>
                  <p>{item.body}</p>
                  {item.canDelete ? (
                    pending === item.id ? (
                      <span className="comment-confirm">
                        <button className="danger" type="button" onClick={() => remove(item.id)}>Удалить комментарий</button>
                        <button type="button" onClick={() => setPending(null)}>Отмена</button>
                      </span>
                    ) : (
                      <button type="button" onClick={() => setPending(item.id)}>Удалить</button>
                    )
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        )
      ) : null}
      {!hidden && pages > 1 ? (
        <nav className="comment-pages" aria-label="Страницы комментариев">
          <button type="button" disabled={current <= 1} aria-label="Предыдущая страница" onClick={() => setPage(current - 1)}>‹</button>
          {pageWindow(current, pages).map((item, index) => (
            item === "…" ? <span key={`gap-${index}`}>…</span> : (
              <button key={item} type="button" aria-current={item === current ? "page" : undefined} onClick={() => setPage(item)}>{item}</button>
            )
          ))}
          <button type="button" disabled={current >= pages} aria-label="Следующая страница" onClick={() => setPage(current + 1)}>›</button>
        </nav>
      ) : null}
      {note ? <p className="explanation">{note}</p> : null}
    </section>
  );
}

function pageWindow(current, pages) {
  if (pages <= 7) {
    return Array.from({ length: pages }, (_, index) => index + 1);
  }
  const start = Math.max(1, current - 2);
  const end = Math.min(pages, current + 2);
  const list = [];
  if (start > 1) {
    list.push(1);
  }
  if (start > 2) {
    list.push("…");
  }
  for (let index = start; index <= end; index += 1) {
    list.push(index);
  }
  if (end < pages - 1) {
    list.push("…");
  }
  if (end < pages) {
    list.push(pages);
  }
  return list;
}

function commentLength(value) {
  return Array.from(value).length;
}

function clipComment(value) {
  const chars = Array.from(value);
  if (chars.length <= commentLimit) {
    return value;
  }
  return chars.slice(0, commentLimit).join("");
}

function mark(name) {
  const first = Array.from(name || "?")[0] || "?";
  return first.toLocaleUpperCase("ru");
}

function stamp(value) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  const minute = String(parsed.getMinutes()).padStart(2, "0");
  return `${parsed.getDate()} ${months[parsed.getMonth()]} ${parsed.getFullYear()}\u00a0г. в\u00a0${parsed.getHours()}:${minute}`;
}
