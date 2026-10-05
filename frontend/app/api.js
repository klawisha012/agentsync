const inflight = new Map();

export async function api(path, options = {}) {
  const method = String(options.method || "GET").toUpperCase();
  const key = method === "GET" && !options.body ? path : "";
  if (key && inflight.has(key)) {
    return inflight.get(key);
  }
  const promise = request(path, options);
  if (key) {
    inflight.set(key, promise);
    promise.finally(() => {
      if (inflight.get(key) === promise) {
        inflight.delete(key);
      }
    });
  }
  return promise;
}

async function request(path, options) {
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
