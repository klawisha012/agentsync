# Для чайников

Весь путь от нуля до рабочей настройки — по шагам. Каждый шаг ссылается на подробную страницу.

## 1. Установите CLI

```sh
curl -fsSL https://zwarder.ru/api/install.sh | sh
```

На Windows в PowerShell:

```powershell
iwr https://zwarder.ru/api/install.ps1 -useb | iex
```

Откройте новый терминал и проверьте команду: `agentsync help`. Подробно: [Установка CLI](./install.md).

## 2. Войдите в аккаунт

```sh
agentsync login
```

Без входа работают справка, версия, обновление, подключение к `~/.agents` и установка открытых навыков. Публикации и хранилище требуют входа. Подробно: [Вход и машина](./login.md).

## 3. Соберите общую настройку в `~/.agents`

Положите навыки в `~/.agents/skills`, общие правила — в `~/.agents/AGENTS.md`, MCP-серверы — в `~/.agents/mcp.json`. Затем подключите ИИ-агента:

```sh
agentsync claude on
```

Вывод называет, что ИИ-агент получил и что пропущено. Подробно: [agents и подключение](../config/agents.md).

## 4. Сделайте снимок перед изменениями

```sh
agentsync record claude
```

Снимок — точка возврата. Подробно: [Локальные снимки](../config/snapshots.md).

## 5. Опубликуйте настройку

```sh
agentsync push claude
agentsync push agents
```

Первая команда публикует Claude Code, вторая — общий каталог `~/.agents`. Подробно: [Публикация](../config/publish.md).

## 6. Примените чужую публикацию

```sh
agentsync apply Author claude
```

`Author` — имя автора на сайте. Подробно: [Применение](../config/apply.md).

## 7. Возьмите отдельные навыки

```sh
agentsync skills Author claude --version 4 --global --into cursor ru-text
```

Подробно: [Навыки](../config/skills.md).

::: tip Если что-то пошло не так
`agentsync revert claude` возвращает последний снимок Claude Code. `agentsync revert agents` возвращает последний снимок `~/.agents`.
:::
