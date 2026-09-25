export async function api(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  if (options.body) {
    headers["Content-Type"] = "application/json";
  }
  const res = await fetch(`/backend${path}`, {
    ...options,
    headers,
    credentials: "same-origin",
  });
  const text = await res.text();
  let body = null;
  if (text) {
    body = JSON.parse(text);
  }
  return { ok: res.ok, status: res.status, body };
}
