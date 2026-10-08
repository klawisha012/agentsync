const bare = /^[\p{L}\p{N}._@+:/-]+$/u;
const quotable = /^[\p{L}\p{N} ._@+:/-]+$/u;

export function shellArg(value) {
  const text = String(value ?? "");
  if (bare.test(text)) {
    return text;
  }
  if (quotable.test(text)) {
    return `"${text}"`;
  }
  return null;
}

export function shellCommand(parts) {
  const out = [];
  for (const part of parts) {
    const arg = shellArg(part);
    if (arg == null) {
      return "";
    }
    out.push(arg);
  }
  return out.join(" ");
}
