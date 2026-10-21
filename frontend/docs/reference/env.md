# Переменные окружения

## AgentSync

| Переменная | Значение | По умолчанию |
|---|---|---|
| `AGENTSYNC_SERVER` | адрес API | `https://zwarder.ru/api` |
| `AGENTSYNC_ROOT` | каталог с ИИ-агентами и `.agents` | домашний каталог |
| `AGENTSYNC_STATE` | каталог состояния: сессия, имя аккаунта, учёт записей `on`; абсолютный путь | `~/.agentsync` |
| `AGENTSYNC_TOKEN` | токен локального агента (машины) | файл `token` в каталоге состояния |
| `AGENTSYNC_COOKIE` | cookie сессии браузера для своей публикации | сессия из `agentsync login` |
| `AGENTSYNC_ACCOUNT` | имя аккаунта, чья цепочка снимков лежит на машине | файл `account` в каталоге состояния |
| `AGENTSYNC_AUTO_UPDATE` | `0` выключает автоматическую проверку обновлений | проверка включена |

## Домашние папки ИИ-агентов

Эти переменные задают ИИ-агенты, а AgentSync их учитывает: подключение, публикация и установка навыков идут туда, где ИИ-агент реально хранит настройку.

| Переменная | ИИ-агент | Что меняется |
|---|---|---|
| `CLAUDE_CONFIG_DIR` | Claude Code | каталог навыков, `agents/`, `commands/`, `CLAUDE.md`, `settings.json`, `.claude.json` |
| `CODEX_HOME` | Codex | каталог навыков, `prompts/`, `AGENTS.md`, `config.toml` |
| `GROK_HOME` | Grok Build | каталог навыков |
| `HERMES_HOME` | Hermes Agent | каталог навыков |
| `VIBE_HOME` | Mistral Vibe | каталог навыков |
| `AUTOHAND_HOME` | Autohand Code CLI | каталог навыков |
| `XDG_CONFIG_HOME` | ИИ-агенты в `~/.config` (Amp, Goose, OpenCode и др.) | каталог навыков |

## Секреты MCP

Значения [переменных секрета](../kinds/mcp.md#переменные-секрета) `${NAME}` AgentSync ищет сначала в окружении, затем в `~/.agents/.env`.
