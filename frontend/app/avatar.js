export function avatarSrc(name, updated) {
  if (!name || !updated) {
    return "";
  }
  return `/api/accounts/${encodeURIComponent(name)}/avatar?v=${encodeURIComponent(updated)}`;
}

export default function Avatar({ name, updated, letter, className }) {
  const src = avatarSrc(name, updated);
  if (!src) {
    return <span className={className} aria-hidden="true">{letter}</span>;
  }
  return <img className={className} src={src} alt="" />;
}
