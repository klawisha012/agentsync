"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../api";
import Comments from "../comments";

export default function AllCommentsPage() {
  const params = useParams();
  const name = String(params.name || "");
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

  const page = state?.ok ? state.body : null;
  const owner = page ? Object.prototype.hasOwnProperty.call(page, "maskedMail") : false;
  const access = page ? page.commentAccess || "hidden" : "hidden";

  return (
    <main className="all-comments-page">
      <Link className="comments-back" href={`/${encodeURIComponent(name)}`}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M19 12H5M12 19l-7-7 7-7" />
        </svg>
        К странице аккаунта
      </Link>
      <h1>Комментарии {name}</h1>
      {!state ? <p className="hint">Открываем комментарии…</p> : null}
      {state && !state.ok ? <p className="explanation">{state.body?.explanation || "Страница не найдена."}</p> : null}
      {page ? <Comments name={page.name} access={access} owner={owner} fullPage /> : null}
    </main>
  );
}
