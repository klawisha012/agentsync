"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../api";

export default function ConfirmPage() {
  const params = useParams();
  const token = params.token;
  const [state, setState] = useState(null);

  useEffect(() => {
    let gone = false;
    api("/email/confirm", {
      method: "POST",
      body: JSON.stringify({ token }),
    }).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [token]);

  return (
    <section className="card">
      <h1>Почта</h1>
      {state === null ? <p className="lede">Подтверждаем почту…</p> : null}
      {state?.ok ? <p className="lede">Почта подтверждена.</p> : null}
      {state && !state.ok ? <p className="explanation">{state.body?.explanation || "Ссылка из письма недействительна или устарела."}</p> : null}
      <Link className="solid" href="/accounts">К списку аккаунтов</Link>
    </section>
  );
}
