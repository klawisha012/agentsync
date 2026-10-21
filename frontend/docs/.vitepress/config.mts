import { defineConfig } from "vitepress";

// The docs are a static VitePress site served by Next from public/docs at /docs/.
export default defineConfig({
  lang: "ru-RU",
  title: "AgentSync",
  description: "Документация AgentSync: публикация, применение и общий каталог ~/.agents для ИИ-агентов.",
  base: "/docs/",
  outDir: "../public/docs",
  cleanUrls: false,
  appearance: "force-dark",
  lastUpdated: false,
  head: [["meta", { name: "theme-color", content: "#000000" }]],
  themeConfig: {
    siteTitle: "AgentSync",
    nav: [
      { text: "Руководство", link: "/start/quick-start" },
      { text: "Справочник", link: "/reference/cli" },
      { text: "Сайт", link: "https://zwarder.ru/", target: "_self" },
    ],
    sidebar: [
      {
        text: "Начало работы",
        items: [
          { text: "Что такое AgentSync", link: "/" },
          { text: "Для чайников", link: "/start/quick-start" },
          { text: "Установка CLI", link: "/start/install" },
          { text: "Вход и машина", link: "/start/login" },
        ],
      },
      {
        text: "Конфигурации",
        items: [
          { text: "Публикация", link: "/config/publish" },
          { text: "agents и подключение", link: "/config/agents" },
          { text: "Локальные снимки", link: "/config/snapshots" },
          { text: "Применение", link: "/config/apply" },
          { text: "Навыки", link: "/config/skills" },
        ],
      },
      {
        text: "Виды элементов agents",
        items: [
          { text: "Навыки", link: "/kinds/skills" },
          { text: "Субагенты и команды", link: "/kinds/subagents" },
          { text: "Файл инструкций", link: "/kinds/instructions" },
          { text: "MCP-серверы", link: "/kinds/mcp" },
          { text: "Плагины", link: "/kinds/plugins" },
          { text: "Хуки", link: "/kinds/hooks" },
        ],
      },
      {
        text: "Сайт",
        items: [
          { text: "Страница аккаунта", link: "/site/account" },
          { text: "Приватность и комментарии", link: "/site/privacy" },
          { text: "Настройки аккаунта", link: "/site/settings" },
        ],
      },
      {
        text: "Справочник",
        items: [
          { text: "Команды CLI", link: "/reference/cli" },
          { text: "Переменные окружения", link: "/reference/env" },
          { text: "Каталог ИИ-агентов", link: "/reference/catalog" },
          { text: "Что не публикуется", link: "/reference/security" },
          { text: "MCP-сервер agentsync", link: "/reference/mcp-server" },
          { text: "Глоссарий", link: "/reference/glossary" },
        ],
      },
    ],
    socialLinks: [{ icon: "github", link: "https://github.com/klawisha012/agentsync" }],
    outline: { level: [2, 3], label: "На этой странице" },
    docFooter: { prev: "Назад", next: "Дальше" },
    darkModeSwitchLabel: "Тема",
    sidebarMenuLabel: "Разделы",
    returnToTopLabel: "Наверх",
    langMenuLabel: "Язык",
    notFound: {
      title: "Страница не найдена",
      quote: "Такой страницы в документации нет.",
      linkLabel: "На главную документации",
      linkText: "К началу",
    },
    search: {
      provider: "local",
      options: {
        translations: {
          button: { buttonText: "Поиск", buttonAriaLabel: "Поиск по документации" },
          modal: {
            displayDetails: "Показать подробности",
            resetButtonTitle: "Сбросить поиск",
            backButtonTitle: "Закрыть поиск",
            noResultsText: "Ничего не найдено по запросу",
            footer: { selectText: "выбрать", navigateText: "перейти", closeText: "закрыть" },
          },
        },
      },
    },
  },
});
