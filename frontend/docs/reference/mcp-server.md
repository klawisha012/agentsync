# MCP-сервер agentsync

::: warning В разработке
Готовой сборки MCP-сервера в установке пока нет. Установщик ставит только CLI. Сервер можно собрать из исходников.
:::

MCP-сервер `agentsync` даёт ИИ-агенту те же действия, что и CLI, через Model Context Protocol (stdio, протокол `2025-06-18`). ИИ-агент вызывает их как инструменты.

## Инструменты

| Инструмент | Аргументы | Что делает |
|---|---|---|
| `push` | `agent` | публикует переносимую настройку ИИ-агента |
| `record` | `agent` | записывает снимок в хранилище |
| `store` | `agent`, `number` | читает снимок из хранилища |
| `revert` | `agent` | возвращает последний снимок на этой машине |
| `apply` | `author`, `agent` | применяет публикацию автора |
| `skills` | `author`, `agent`, `version`, `skills`, `into`, `place` (`global` или `project`) | ставит выбранные навыки публикации |

## Сборка из исходников

```sh
git clone https://github.com/klawisha012/agentsync
cd agentsync/backend
go build -o agentsync-mcp ./cmd/mcp
```

Сервер читает те же [переменные окружения](./env.md), что и CLI: `AGENTSYNC_ROOT`, `AGENTSYNC_SERVER`, `AGENTSYNC_TOKEN`, `AGENTSYNC_ACCOUNT`, и сессию из `agentsync login`.

## Подключение к Claude Code

```sh
claude mcp add --scope user agentsync -- /path/to/agentsync-mcp
```

Другие ИИ-агенты подключают stdio-сервер по своим правилам. Если сервер описан в `~/.agents/mcp.json`, `agentsync <ии-агент> on` впишет его в настройки всех подключённых ИИ-агентов: [MCP-серверы](../kinds/mcp.md).

::: tip Машинный путь
Абсолютный путь к бинарнику делает сервер [машинным MCP](../kinds/mcp.md#машинный-mcp): он работает на этой машине, но в публикацию не входит.
:::
