"use client";

import { useEffect, useState } from "react";
import { Command } from "./command";

const NAV = [
  { group: "Для чайников", items: [{ id: "beginners", label: "Для чайников" }] },
  { group: "Начало работы", items: [
    { id: "installation", label: "Установка CLI" },
    { id: "login", label: "Вход" },
  ] },
  { group: "Конфигурации", items: [
    { id: "publish", label: "Публикация" },
    { id: "snapshots", label: "Локальные снимки" },
    { id: "apply", label: "Применение" },
    { id: "skills", label: "Навыки" },
  ] },
  { group: "Справочник", items: [
    { id: "help", label: "Справка по командам" },
    { id: "mcp", label: "MCP", soon: true },
  ] },
];
const FLAT = NAV.flatMap((group) => group.items.map((item) => ({ ...item, group: group.group })));

export default function DocsPage() {
  const [active, setActive] = useState("beginners");
  const [query, setQuery] = useState("");

  useEffect(() => {
    const onScroll = () => {
      let current = "beginners";
      for (const item of FLAT) {
        const el = document.getElementById(item.id);
        if (el && el.getBoundingClientRect().top < 160) {
          current = item.id;
        }
      }
      setActive(current);
    };
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const index = FLAT.findIndex((item) => item.id === active);
  const current = FLAT[index] || FLAT[0];
  const q = query.trim().toLowerCase();

  return (
    <div className="vdocs">
      <aside className="vdocs-side" aria-label="Разделы документации">
        <label className="vdocs-search">
          <Search />
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Поиск по документации" />
        </label>
        {NAV.map((group) => {
          const items = group.items.filter((item) => !q || item.label.toLowerCase().includes(q));
          if (!items.length) {
            return null;
          }
          return (
            <div className="vdocs-group" key={group.group}>
              <span className="vdocs-group-title">{group.group}</span>
              {items.map((item) => (
                <a key={item.id} href={`#${item.id}`} className={active === item.id ? "is-active" : undefined}>
                  {item.label}{item.soon ? <span className="vdocs-soon">скоро</span> : null}
                </a>
              ))}
            </div>
          );
        })}
      </aside>
      <article className="docs-page vdocs-main">
        <nav className="vdocs-crumbs" aria-label="Путь">
          <Book /> Документация <Chevron /> {current.group} <Chevron /> <span>{current.label}</span>
        </nav>
        <header className="docs-heading">
          <h1>AgentSync CLI и MCP</h1>
          <p>Установка, публикация конфигураций и работа с локальными снимками через командную строку.</p>
        </header>
        <div className="docs-content">
          <section className="docs-section" id="beginners">
            <h2><Compass /> Для чайников</h2>
            <p>Весь путь от нуля до рабочей конфигурации — по шагам. Каждый шаг ссылается на подробный раздел ниже.</p>
            <ol className="docs-steps">
              <Step n="1" title="Установите CLI" href="#installation" link="Установка CLI" code="curl -fsSL https://zwarder.ru/api/install.sh | sh">
                Скачайте и запустите установщик для своей системы. Команда появится в терминале.
              </Step>
              <Step n="2" title="Войдите в аккаунт" href="#login" link="Вход" code="agentsync login">
                Без входа работает только просмотр справки. Почту можно указать сразу: agentsync login alice@example.com.
              </Step>
              <Step n="3" title="Опубликуйте свою конфигурацию" href="#publish" link="Публикация" code="agentsync push Grok">
                Отправьте настройку своего AI-агента на сервер, чтобы ею могли пользоваться другие. Grok в команде — пример имени.
              </Step>
              <Step n="4" title="Делайте снимки перед изменениями" href="#snapshots" link="Локальные снимки" code="agentsync record Grok">
                Снимок — точка возврата. Создайте его перед любым рискованным действием.
              </Step>
              <Step n="5" title="Примените чужую публикацию" href="#apply" link="Применение" code="agentsync apply Author Grok">
                Нашли интересную конфигурацию? Примените её по имени автора и агента. Author — пример имени автора.
              </Step>
              <Step n="6" title="Установите нужные навыки" href="#skills" link="Навыки" code="agentsync skills Author Grok --version 4 --global --into cursor ru-text">
                Из любой публикации можно взять отдельные навыки и установить их в свой инструмент.
              </Step>
            </ol>
            <div className="docs-notice"><p>Коротко: установили → вошли → опубликовали → сделали снимок → применили чужое → добавили навыки. Если что-то пошло не так — <code>agentsync revert Grok</code> вернёт последний снимок.</p></div>
          </section>
          <section className="docs-section" id="installation">
            <h2><Terminal /> Установка CLI</h2>
            <p>CLI — отдельная программа для терминала. Установщик помещает её в <code>~/.agentsync</code> и добавляет в <code>PATH</code>.</p>
            <div className="docs-notice"><p>Перед запуском прочитайте установочный скрипт: он выполняет загруженный код на вашем компьютере.</p></div>
            <h3>Linux / macOS</h3>
            <Command code="curl -fsSL https://zwarder.ru/api/install.sh | sh" />
            <h3>Windows · PowerShell</h3>
            <Command code="iwr https://zwarder.ru/api/install.ps1 -useb | iex" />
            <p>Откройте новый терминал после установки и проверьте доступность команды.</p>
            <Command code="agentsync help" />
            <a className="docs-reference" href="https://github.com/klawisha012/agentsync/tree/main/scripts" target="_blank" rel="noopener noreferrer">Исходные скрипты установки <External /></a>
          </section>
          <section className="docs-section" id="login">
            <h2>Вход</h2>
            <p>Войдите перед публикацией. Укажите почту своего аккаунта; <code>alice@example.com</code> ниже — только пример.</p>
            <Command code="agentsync login alice@example.com" />
            <p>Можно запустить <code>agentsync login</code> без почты. Сессия пишется в <code>~/.agentsync/session</code>. Пароль на диск не записывается. Не публикуйте этот файл.</p>
          </section>
          <section className="docs-section" id="publish">
            <h2>Публикация конфигурации</h2>
            <p><code>push</code> отправляет переносимую конфигурацию выбранного AI-агента на сервер. Учётные данные в запрос не входят.</p>
            <Command code="agentsync push Grok" />
            <p>В примерах используется <code>Grok</code>. Подставьте имя своего агента и проверьте файлы перед отправкой.</p>
          </section>
          <section className="docs-section" id="snapshots">
            <h2>Локальные снимки</h2>
            <h3>Сохранить текущее состояние</h3>
            <p><code>record</code> сохраняет снимок в <code>~/.agentsync/store</code> и на сервере. Публикацию эта команда не создаёт.</p>
            <Command code="agentsync record Grok" />
            <h3>Посмотреть сохранённые снимки</h3>
            <Command code="agentsync store Grok" />
            <p>Добавьте номер, чтобы посмотреть содержимое конкретного снимка.</p>
            <Command code="agentsync store Grok 2" />
            <h3>Вернуться к последнему снимку</h3>
            <Command code="agentsync revert Grok" />
            <p>Возврат изменяет текущие файлы конфигурации. Сначала сохраните важные изменения. Публикации на сервере не меняются.</p>
          </section>
          <section className="docs-section" id="apply">
            <h2>Применение публикации</h2>
            <p>Передайте имя автора и AI-агента. <code>Author</code> — пример имени автора; замените его реальным.</p>
            <Command code="agentsync apply Author Grok" />
            <div className="docs-notice"><p><code>apply</code> заменяет текущую переносимую настройку файлами из публикации. Перед заменой текущая настройка записывается в снимок. Откат: <code>agentsync revert Grok</code>.</p></div>
          </section>
          <section className="docs-section" id="skills">
            <h2>Установка отдельных навыков</h2>
            <p><code>skills</code> копирует выбранные навыки из конкретной версии публикации в каталоги выбранных приёмников.</p>
            <Command code="agentsync skills Author Grok --version 4 --global --into cursor ru-text" />
            <ul>
              <li><code>Author Grok</code> — автор и агент.</li>
              <li><code>--version 4</code> — номер опубликованной версии.</li>
              <li><code>--global</code> — глобальная установка; <code>--project</code> — установка для текущего проекта.</li>
              <li><code>--into cursor</code> — приёмник. Флаг повторяется для каждого приёмника.</li>
              <li><code>ru-text</code> — пример имени навыка; укажите имя из выбранной публикации.</li>
            </ul>
            <p>Без <code>--into</code> и без <code>--global</code> или <code>--project</code> команда сначала спрашивает приёмников, затем место установки.</p>
          </section>
          <section className="docs-section" id="help">
            <h2>Справка по командам</h2>
            <p>Для подробностей передайте имя команды после <code>help</code>.</p>
            <Command code="agentsync help push" />
            <Command code="agentsync help skills" />
            <a className="docs-reference" href="https://github.com/klawisha012/agentsync" target="_blank" rel="noopener noreferrer">Исходный проект AgentSync <External /></a>
          </section>
          <section className="docs-section" id="mcp">
            <h2><Plug /> MCP <span className="docs-status"><Clock /> В разработке</span></h2>
            <p>Model Context Protocol — способ подключения инструментов к AI-клиентам. Готовой инструкции подключения на сайте пока нет.</p>
            <p>После публикации здесь появятся требования, настройки подключения, список доступных инструментов и примеры использования.</p>
            <div className="docs-notice"><p>Этот раздел зарезервирован для будущей документации MCP.</p></div>
          </section>
        </div>
      </article>
      <aside className="vdocs-toc" aria-label="На этой странице">
        <span className="vdocs-group-title">На этой странице</span>
        {FLAT.map((item) => (
          <a key={item.id} href={`#${item.id}`} className={active === item.id ? "is-active" : undefined}>{item.label}</a>
        ))}
      </aside>
    </div>
  );
}

function Step({ n, title, href, link, code, children }) {
  return (
    <li>
      <span className="docs-step-num">{n}</span>
      <div>
        <h3>{title}</h3>
        <p>{children}</p>
        <Command code={code} />
        <p>Подробности — в разделе <a href={href}>{link}</a>.</p>
      </div>
    </li>
  );
}

function Icon({ children, size = 16 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {children}
    </svg>
  );
}

function Search() {
  return <Icon><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></Icon>;
}

function Book() {
  return <Icon size={14}><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" /><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" /></Icon>;
}

function Chevron({ dir }) {
  const d = dir === "left" ? "M15 6 9 12l6 6" : "M9 6l6 6-6 6";
  return <Icon size={13}><path d={d} /></Icon>;
}

function Compass() {
  return <Icon size={20}><circle cx="12" cy="12" r="9" /><path d="m16 8-2.5 6.5L8 16l2.5-6.5z" /></Icon>;
}

function Terminal() {
  return <Icon size={20}><path d="m4 5 4 4-4 4M10 17h8" /><rect x="2" y="3" width="20" height="18" rx="2" /></Icon>;
}

function Plug() {
  return <Icon size={20}><path d="M12 22v-5M9 8V2M15 8V2M7 8h10v5a5 5 0 0 1-10 0z" /></Icon>;
}

function Clock() {
  return <Icon size={12}><circle cx="12" cy="12" r="9" /><path d="M12 7v6l4 2" /></Icon>;
}

function External() {
  return <Icon size={13}><path d="M14 4h6v6M20 4 10 14" /><path d="M18 13v6a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h6" /></Icon>;
}
