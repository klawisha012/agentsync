import Link from "next/link";

export default function Home() {
  return (
    <section>
      <h1>AgentSync</h1>
      <p className="lede" style={{ textAlign: "left" }}>
        Публикация переносимой настройки одного ИИ-агента. Вход и страница аккаунта уже открыты.
      </p>
      <Link className="solid" href="/login?tab=register">
        Создать аккаунт
      </Link>
    </section>
  );
}
