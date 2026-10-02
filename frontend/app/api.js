export async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body) {
    headers["Content-Type"] = "application/json";
  }
  const res = await fetch(`/api${path}`, {
    ...options,
    headers,
    credentials: "same-origin",
  });
  const text = await res.text();
  let body = null;
  let parsed = true;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      parsed = false;
      body = { explanation: "Сервер не ответил данными." };
    }
  }
  return { ok: res.ok && parsed, status: res.status, body };
}
