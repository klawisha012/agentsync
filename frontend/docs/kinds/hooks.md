# Хуки

Хук — команда shell, которую ИИ-агент запускает на событии: перед вызовом инструмента, после него, в начале сессии и так далее. Хуки `~/.agents` лежат в `~/.agents/hooks.json` в формате agents: события и инструменты называются словами agents, а не именами одного ИИ-агента.

```json
{
  "version": 1,
  "hooks": {
    "before_tool": [
      {
        "matcher": ["shell"],
        "input": "claude",
        "hooks": [{ "type": "command", "command": "\"${AGENTS_PROJECT_DIR}\"/hooks/check.sh", "timeout": 10 }]
      }
    ],
    "stop": [
      { "hooks": [{ "type": "command", "command": "say done > /dev/null" }] }
    ]
  },
  "claude": {
    "hooks": {
      "PermissionRequest": [{ "hooks": [{ "type": "command", "command": "ask.sh" }] }]
    }
  }
}
```

- `"version": 1` обязателен. Без него `on`, `record agents` и `push agents` отказывают: `Хуки в ~/.agents/hooks.json в старом формате…`.
- Запись хука — одна группа `{matcher, input, hooks}` внутри одного события.
- `matcher` — список инструментов словаря или `"*"`. Без `matcher` хук срабатывает на любой инструмент. `matcher` бывает только у событий инструментов: `before_tool`, `after_tool`, `after_tool_failure`.
- `input` — [формат входа хука](#формат-входа): в каком формате скрипт читает данные события.
- `timeout` — в секундах.
- `${AGENTS_PROJECT_DIR}` в команде — каталог проекта. `on` подставляет родную переменную ИИ-агента, а если её нет — `${PWD}`.
- Раздел расширений — ключ с именем ИИ-агента из [каталога](../reference/catalog.md) (`claude`, `gemini`…). Внутри — хуки в родном формате этого ИИ-агента. Их получает только он.

## Словарь

| События | |
|---|---|
| `session_start`, `session_end` | начало и конец сессии |
| `prompt_submit` | человек отправил запрос |
| `before_tool`, `after_tool`, `after_tool_failure` | до инструмента, после него, после его ошибки |
| `stop` | ИИ-агент закончил ответ |
| `subagent_start`, `subagent_stop` | начало и конец работы субагента |
| `notification` | уведомление |
| `pre_compact`, `post_compact` | до и после сжатия контекста |

Инструменты: `shell`, `read`, `write`, `edit`, `glob`, `grep`, `list`, `web_fetch`, `web_search`, `task` (вызов субагента). Инструмент MCP: `mcp:<сервер>/<инструмент>`, все инструменты сервера: `mcp:<сервер>/*`.

## Подключение

| ИИ-агент | Куда вписываются хуки |
|---|---|
| Claude Code | `~/.claude/settings.json`, ключ `hooks` |
| Gemini CLI | `~/.gemini/settings.json`, ключ `hooks`; `timeout` в миллисекундах |
| Grok Build | `~/.grok/hooks/agentsync.json` |
| Codex | `~/.codex/hooks.json` |
| GitHub Copilot | `~/.copilot/hooks/agentsync.json`: одна запись `{type, matcher, bash, timeoutSec}` на каждую команду |
| Остальные | вид «хуки» пропускается |

Cursor хуки пока не получает: место его хуков не подтверждено.

| Слово agents | Claude Code | Gemini CLI | Grok Build | Codex | GitHub Copilot |
|---|---|---|---|---|---|
| `session_start` | `SessionStart` | `SessionStart` | `SessionStart` | `SessionStart` | `sessionStart` |
| `session_end` | `SessionEnd` | `SessionEnd` | `SessionEnd` | — | `sessionEnd` |
| `prompt_submit` | `UserPromptSubmit` | `BeforeAgent` | `UserPromptSubmit` | `UserPromptSubmit` | `userPromptSubmitted` |
| `before_tool` | `PreToolUse` | `BeforeTool` | `PreToolUse` | `PreToolUse` | `preToolUse` |
| `after_tool` | `PostToolUse` | `AfterTool` | `PostToolUse` | `PostToolUse` | `postToolUse` |
| `after_tool_failure` | `PostToolUseFailure` | — | `PostToolUseFailure` | — | `postToolUseFailure` |
| `stop` | `Stop` | `AfterAgent` | `Stop` | `Stop` | `agentStop` |
| `subagent_start`, `subagent_stop` | `SubagentStart`, `SubagentStop` | — | `SubagentStart`, `SubagentStop` | — | `subagentStart`, `subagentStop` |
| `notification` | `Notification` | `Notification` | `Notification` | — | `notification` |
| `pre_compact` | `PreCompact` | `PreCompress` | `PreCompact` | — | `preCompact` |
| `post_compact` | — | — | `PostCompact` | — | — |
| `shell` | `Bash` | `run_shell_command` | `run_terminal_command` | `Bash` | `bash` |
| `read` | `Read` | `read_file` | `read_file` | — | `view` |
| `write` | `Write` | `write_file` | `search_replace` | `apply_patch` | `create` |
| `edit` | `Edit`, `MultiEdit` | `replace` | `search_replace` | `apply_patch` | `edit` |
| `glob` | `Glob` | `glob` | — | — | `glob` |
| `grep` | `Grep` | `search_file_content` | `grep` | — | `grep` |
| `list` | `LS` | `list_directory` | `list_dir` | — | — |
| `web_fetch` | `WebFetch` | `web_fetch` | — | — | `web_fetch` |
| `web_search` | `WebSearch` | `google_web_search` | `web_search` | — | — |
| `task` | `Task` | — | `spawn_subagent` | — | `task` |
| `mcp:github/*` | `mcp__github__.*` | `mcp_github_.*` | `github__.*` | `mcp__github__.*` | `github/.*` |
| `${AGENTS_PROJECT_DIR}` | `${CLAUDE_PROJECT_DIR}` | `${GEMINI_PROJECT_DIR}` | `${PWD}` | `${PWD}` | `${PWD}` |

«—» значит, что у ИИ-агента такого события или инструмента нет:

- хук события, которого у ИИ-агента нет, ему не достаётся, и вывод `on` называет событие: `хуки событий subagent_stop`;
- инструмент, которого у ИИ-агента нет, выпадает из `matcher` только у этого ИИ-агента: `инструменты хуков task`;
- если после этого в `matcher` не осталось ни одного инструмента, хук этому ИИ-агенту не достаётся. Пустой список никогда не значит «все инструменты».

::: warning Проверьте перевод
Имена событий и инструментов меняются между версиями ИИ-агентов. Если хук не срабатывает, сверьте их с документацией вашей версии.
:::

## Отказ подключения

Эти причины проверяются до изменения файлов.

| Пояснение | Что сделать |
|---|---|
| `…в старом формате…` | Добавьте `"version": 1` и замените имена событий и инструментов словами agents. |
| `событие «…» не из словаря agents` | Возьмите слово из [словаря](#словарь) или перенесите хук в раздел расширений своего ИИ-агента. |
| `инструмент «…» не из словаря agents` | То же для инструмента. |
| `Matcher … есть только у событий инструментов` | Уберите `matcher` или перенесите хук в раздел расширений. |

## Формат входа

Скрипт хука читает данные события из stdin. Поле `input` говорит, в каком формате: имя ИИ-агента (`claude`, `gemini`…), `agents` или `none`, если скрипт stdin не читает.

## Свои и чужие записи

Правила те же, что у [MCP-серверов](./mcp.md#свои-и-чужие-записи): чужие хуки остаются, повторный `on` убирает свой удалённый хук, `off` удаляет только свои. Одинаковые хуки сравниваются по содержанию, поэтому повторный `on` не дублирует хук.

Публикация ИИ-агента помнит, из какой записи `~/.agents` пришёл каждый вписанный хук. При применении такой публикации в `~/.agents` ложится именно эта запись, а не обратный перевод.

## Машинный хук

Хук, в команде которого есть абсолютный путь (`/Users/me/run.sh`, `C:\scripts\x.ps1`), привязан к машине.

- Машинный хук в снимок и публикацию не входит — ни сам хук, ни скрипт, на который он ссылается.
- Применение, откат и подключение машинные хуки этой машины не убирают.
- `/dev/null` и путь под переменной (`"${AGENTS_PROJECT_DIR}"/hooks/x.sh`, `$HOME/…`) машинным путём не считаются: такой хук публикуется.

Файл хуков (`hooks.json`, каталог `hooks/`), где абсолютный путь стоит вне раздела `hooks` (например, в `env` или в разделе расширений), в публикацию не входит целиком.
