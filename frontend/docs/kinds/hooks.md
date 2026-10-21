# Хуки

Хук — команда shell, которую ИИ-агент запускает на событии: перед вызовом инструмента, после него, в начале сессии и так далее. Хуки `~/.agents` лежат в `~/.agents/hooks.json` в формате событий Claude Code.

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{ "type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/check.sh" }]
      }
    ],
    "Stop": [
      { "hooks": [{ "type": "command", "command": "say done > /dev/null" }] }
    ]
  }
}
```

Запись хука — одна группа `{matcher, hooks}` внутри одного события.

## Подключение

| ИИ-агент | Куда вписываются хуки | Перевод |
|---|---|---|
| Claude Code | `~/.claude/settings.json`, ключ `hooks` | без перевода |
| Gemini CLI | `~/.gemini/settings.json`, ключ `hooks` | события и имена инструментов |
| Остальные | — | вид «хуки» пропускается |

### Перевод для Gemini CLI

| Событие Claude Code | Событие Gemini CLI |
|---|---|
| `PreToolUse` | `BeforeTool` |
| `PostToolUse` | `AfterTool` |
| `UserPromptSubmit` | `BeforeAgent` |
| `Stop` | `AfterAgent` |
| `SessionStart` | `SessionStart` |
| `SessionEnd` | `SessionEnd` |
| `PreCompact` | `PreCompress` |
| `Notification` | `Notification` |

| Инструмент Claude Code в `matcher` | Инструмент Gemini CLI |
|---|---|
| `Bash` | `run_shell_command` |
| `Write` | `write_file` |
| `Edit`, `MultiEdit` | `replace` |
| `Read` | `read_file` |
| `Glob` | `glob` |
| `Grep` | `search_file_content` |
| `LS` | `list_directory` |
| `WebFetch` | `web_fetch` |
| `WebSearch` | `google_web_search` |

Хук события, которого у ИИ-агента нет (например, `PermissionRequest` у Gemini CLI), этому ИИ-агенту не достаётся. Вывод `on` называет такие события: `Пропущены виды, которых ИИ-агент не понимает: хуки событий PermissionRequest.`

::: warning Проверьте перевод
Имена событий и инструментов Gemini CLI меняются между версиями. Если хук не срабатывает в Gemini CLI, сверьте их с документацией вашей версии.
:::

### Свои и чужие записи

Правила те же, что у [MCP-серверов](./mcp.md#свои-и-чужие-записи): чужие хуки остаются, повторный `on` убирает свой удалённый хук, `off` удаляет только свои. Одинаковые хуки сравниваются по содержанию, поэтому повторный `on` не дублирует хук.

## Машинный хук

Хук, в команде которого есть абсолютный путь (`/Users/me/run.sh`, `C:\scripts\x.ps1`), привязан к машине.

- Машинный хук в снимок и публикацию не входит — ни сам хук, ни скрипт, на который он ссылается.
- Применение, откат и подключение машинные хуки этой машины не убирают.
- `/dev/null` и путь под переменной (`"$CLAUDE_PROJECT_DIR"/.claude/hooks/x.sh`, `$HOME/…`) машинным путём не считаются: такой хук публикуется.

Файл хуков (`hooks.json`, каталог `hooks/`), где абсолютный путь стоит вне раздела `hooks` (например, в `env`), в публикацию не входит целиком.
