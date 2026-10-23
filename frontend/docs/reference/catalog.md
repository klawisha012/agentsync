# Каталог ИИ-агентов

Каталог ИИ-агентов поставляется вместе с AgentSync. Для каждого ИИ-агента он называет имя, синонимы и места, куда ложатся элементы `~/.agents`. Команды `on`, `off` и `skills` работают с любым ИИ-агентом каталога. Карточка на сайте появляется только после публикации.

Имя в команде — имя из каталога или его синоним, без учёта регистра: `agentsync claude on` и `agentsync claude-code on` равны.

## Виды кроме навыков

| ИИ-агент | Субагенты | Команды | Файл инструкций | MCP-серверы | Плагины | Хуки |
|---|---|---|---|---|---|---|
| Claude Code | `agents/` | `commands/` | `CLAUDE.md` | `~/.claude.json` | `settings.json` | `settings.json` |
| Codex | `agents/`, TOML | — | `AGENTS.md` | `config.toml` | — | `hooks.json` |
| Gemini CLI | `agents/` | `commands/`, TOML | `GEMINI.md` | `settings.json` | — | `settings.json` |
| Grok Build | `agents/` | читает сам | — | `config.toml` | — | `hooks/agentsync.json` |
| GitHub Copilot | `agents/*.agent.md` | — | — | — | — | `hooks/agentsync.json` |
| Cursor | `agents/`, только readonly | — | — | — | — | — |
| Остальные | — | — | — | — | — | — |

Пути — относительно домашней папки ИИ-агента (`~/.claude`, `~/.codex`, `~/.gemini`). «—» значит, что вид пропускается и вывод `on` его называет.

## Навыки

Глобальный каталог навыков — место, куда `on` ставит ссылки на навыки `~/.agents`, а `skills --global` кладёт навыки публикации. `$ПЕРЕМЕННАЯ` после пути — переменная окружения, которая переносит каталог ИИ-агента. Для ИИ-агентов в `~/.config` каталог переносит `XDG_CONFIG_HOME`.

| ��-����� | ��� � ������� | ���������� ������� ������� |
|---|---|---|
| AdaL | `adal` | `~/.adal/skills` |
| AiderDesk | `aider-desk` | `~/.aider-desk/skills` |
| Amp | `amp` | `~/.config/agents/skills` |
| Antigravity | `antigravity` | `~/.gemini/antigravity/skills` |
| Antigravity CLI | `antigravity-cli` | `~/.gemini/antigravity-cli/skills` |
| AstrBot | `astrbot` | `~/.astrbot/data/skills` |
| Augment | `augment` | `~/.augment/skills` |
| Autohand Code CLI | `autohand-code` | `~/.autohand/skills`, `$AUTOHAND_HOME/skills` |
| Claude Code | `claude-code`, `claude` | `~/.claude/skills`, `$CLAUDE_CONFIG_DIR/skills` |
| Cline | `cline` | `~/.agents/skills` (������ agents ��������) |
| Code Studio | `codestudio` | `~/.codestudio/skills` |
| CodeArts Agent | `codearts-agent` | `~/.codeartsdoer/skills` |
| CodeBuddy | `codebuddy` | `~/.codebuddy/skills` |
| Codemaker | `codemaker` | `~/.codemaker/skills` |
| Codex | `codex` | `~/.codex/skills`, `$CODEX_HOME/skills` |
| Command Code | `command-code` | `~/.commandcode/skills` |
| Continue | `continue` | `~/.continue/skills` |
| Cortex Code | `cortex` | `~/.snowflake/cortex/skills` |
| Crush | `crush` | `~/.config/crush/skills` |
| Cursor | `cursor` | `~/.cursor/skills` |
| Deep Agents | `deepagents` | `~/.deepagents/agent/skills` |
| Devin for Terminal | `devin` | `~/.config/devin/skills` |
| Dexto | `dexto` | `~/.agents/skills` (������ agents ��������) |
| Droid | `droid` | `~/.agents/skills` (������ agents ��������) |
| Eve | `eve` | � |
| Firebender | `firebender` | `~/.firebender/skills` |
| ForgeCode | `forgecode` | `~/.forge/skills` |
| fx | `fx` | `~/.fx/skills` |
| Gemini CLI | `gemini-cli`, `gemini` | `~/.gemini/skills` |
| GitHub Copilot | `github-copilot`, `copilot` | `~/.copilot/skills` |
| Goose | `goose` | `~/.config/goose/skills` |
| Grok Build | `grok` | `~/.grok/skills`, `$GROK_HOME/skills` |
| Hermes Agent | `hermes-agent` | `~/.hermes/skills`, `$HERMES_HOME/skills` |
| IBM Bob | `bob` | `~/.bob/skills` |
| iFlow CLI | `iflow-cli` | `~/.iflow/skills` |
| inference.sh | `inference-sh` | `~/.inferencesh/skills` |
| Jazz | `jazz` | `~/.jazz/skills` |
| Junie | `junie` | `~/.junie/skills` |
| Kilo Code | `kilo` | `~/.kilo/skills` |
| Kimchi | `kimchi` | `~/.config/kimchi/harness/skills` |
| Kimi Code CLI | `kimi-code-cli` | `~/.agents/skills` (������ agents ��������) |
| Kiro CLI | `kiro-cli` | `~/.kiro/skills` |
| Kode | `kode` | `~/.kode/skills` |
| Lingma | `lingma` | `~/.lingma/skills` |
| Loaf | `loaf` | `~/.agents/skills` (������ agents ��������) |
| MCPJam | `mcpjam` | `~/.mcpjam/skills` |
| MiniMax Code | `minimax-code` | `~/.minimax/skills` |
| Mistral Vibe | `mistral-vibe` | `~/.vibe/skills`, `$VIBE_HOME/skills` |
| Moxby | `moxby` | `~/.moxby/skills` |
| Mux | `mux` | `~/.mux/skills` |
| Neovate | `neovate` | `~/.neovate/skills` |
| Ona | `ona` | `~/.ona/skills` |
| OpenClaw | `openclaw` | `~/.openclaw/skills` |
| OpenCode | `opencode` | `~/.config/opencode/skills` |
| OpenHands | `openhands` | `~/.openhands/skills` |
| Pi | `pi` | `~/.agents/skills` (������ agents ��������) |
| Pochi | `pochi` | `~/.pochi/skills` |
| Posit Assistant | `posit-assistant` | `~/.posit/assistant/skills` |
| PromptScript | `promptscript` | � |
| Qoder | `qoder` | `~/.qoder/skills` |
| Qoder CN | `qoder-cn` | `~/.qoder-cn/skills` |
| Qwen Code | `qwen-code` | `~/.qwen/skills` |
| Reasonix | `reasonix` | `~/.reasonix/skills` |
| Replit | `replit` | `~/.config/agents/skills` |
| Roo Code | `roo` | `~/.roo/skills` |
| Rovo Dev | `rovodev` | `~/.rovodev/skills` |
| Sarvam Code | `sarvam-code` | `~/.agents/skills` (������ agents ��������) |
| Tabnine CLI | `tabnine-cli` | `~/.tabnine/agent/skills` |
| Terramind | `terramind` | `~/.terramind/skills` |
| Tinycloud | `tinycloud` | `~/.tinycloud/skills` |
| Trae | `trae` | `~/.trae/skills` |
| Trae CN | `trae-cn` | `~/.trae-cn/skills` |
| Warp | `warp` | `~/.agents/skills` (������ agents ��������) |
| Windsurf | `windsurf` | `~/.codeium/windsurf/skills` |
| ZCode | `zcode` | `~/.zcode/skills` |
| Zed | `zed` | `~/.agents/skills` (������ agents ��������) |
| Zencoder | `zencoder` | `~/.zencoder/skills` |
| Zenflow | `zenflow` | `~/.zencoder/skills` |

ИИ-агенты с пометкой «читает agents напрямую» уже видят навыки `~/.agents/skills`. Для них `on` навыки не трогает, а `off` отказывает.

ИИ-агент без глобального каталога («—») поддерживает только установку в проект: `agentsync skills … --project`.
