"use client";

import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "../../api";

export default function PublicationPage() {
  const params = useParams();
  const [state, setState] = useState(null);

  useEffect(() => {
    let gone = false;
    api(`/publications/${encodeURIComponent(params.id)}`).then((res) => {
      if (!gone) {
        setState(res);
      }
    });
    return () => {
      gone = true;
    };
  }, [params.id]);

  if (!state) {
    return <p className="lede">Открываем публикацию…</p>;
  }
  if (!state.ok) {
    return <p className="explanation">{state.body?.explanation || "Публикация не найдена."}</p>;
  }
  const files = state.body.files || [];
  return (
    <section>
      <h1>{state.body.agent} v{state.body.version}</h1>
      <p className="hint">Постоянный адрес этой публикации. Предпросмотр файлов откроется отдельно.</p>
      <ul className="policy">
        {files.map((file) => (
          <li key={file.path}><code>{file.path}</code></li>
        ))}
      </ul>
    </section>
  );
}
