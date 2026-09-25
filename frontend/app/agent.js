"use client";

import { useCallback, useEffect, useState } from "react";

export const listenerLabel = "127.0.0.1:49152";

export async function probeAgent() {
  const started = performance.now();
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), 1500);
  try {
    const res = await fetch("/agent-bridge/status", { signal: ctrl.signal, cache: "no-store" });
    const ms = Math.max(1, Math.round(performance.now() - started));
    if (!res.ok) {
      return { ok: false };
    }
    const body = await res.json();
    return {
      ok: true,
      ms,
      id: body.id || "",
      host: body.host || "",
      account: body.account || "",
    };
  } catch {
    return { ok: false };
  } finally {
    clearTimeout(timer);
  }
}

export async function handToken(token) {
  try {
    const res = await fetch("/agent-bridge/session", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    });
    return res.ok;
  } catch {
    return false;
  }
}

export async function dropAgent() {
  try {
    await fetch("/agent-bridge/session", { method: "DELETE" });
  } catch {
    // Агент уже молчит: снятие на сервере всё равно выводит его из аккаунта.
  }
}

export function useAgentProbe() {
  const [state, setState] = useState({ ok: false, pending: true });
  const refresh = useCallback(async () => {
    const next = await probeAgent();
    setState({ ...next, pending: false });
    return next;
  }, []);

  useEffect(() => {
    let gone = false;
    const pull = () => {
      probeAgent().then((next) => {
        if (!gone) {
          setState({ ...next, pending: false });
        }
      });
    };
    pull();
    const id = setInterval(pull, 5000);
    return () => {
      gone = true;
      clearInterval(id);
    };
  }, []);

  return [state, refresh];
}
