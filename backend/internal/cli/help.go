package cli

import (
	"fmt"
	"strings"
)

type envLine struct {
	name string
	text string
}

type spec struct {
	name     string
	summary  string
	usage    string
	about    string
	envs     []envLine
	examples []string
	minArgs  int
	missing  string
}

func commands() []spec {
	server := envLine{name: "AGENTSYNC_SERVER", text: "Адрес API"}
	root := envLine{name: "AGENTSYNC_ROOT", text: "Каталог с\u00a0ИИ-агентами"}
	token := envLine{name: "AGENTSYNC_TOKEN", text: "Токен локального агента"}
	cookie := envLine{name: "AGENTSYNC_COOKIE", text: "Cookie сессии браузера для своей публикации"}
	account := envLine{name: "AGENTSYNC_ACCOUNT", text: "Имя аккаунта, чья цепочка лежит на машине"}
	return []spec{
		{
			name:    "login",
			summary: "Входит в аккаунт и сохраняет сессию в домашнем каталоге",
			usage:   "agentsync login [почта]",
			about: "Сессия пишется в ~/.agentsync/session. В репозиторий она не попадает.\n" +
				"Пароль на диск не записывается.",
			examples: []string{"agentsync login"},
		},
		{
			name:    "push",
			summary: "Публикует переносимую настройку ~/.agents",
			usage:   "agentsync push",
			about: "На сервер уходят только переносимые файлы ~/.agents под именем agents.\n" +
				"Файл ~/.agents/.env, учётные данные, машинные MCP и\u00a0машинные хуки в\u00a0публикацию не\u00a0входят.\n" +
				"Чтобы сначала собрать настройки установленных ИИ-агентов в\u00a0~/.agents, запустите agentsync export.\n" +
				"Нужен вход в\u00a0аккаунт. Подтверждение почты для публикации не\u00a0нужно.\n" +
				"Пароль в\u00a0команду не\u00a0входит.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync push"},
		},
		{
			name:    "record",
			summary: "Записывает снимок ~/.agents в\u00a0хранилище",
			usage:   "agentsync record",
			about: "Снимок неизменяемый. Команда печатает ответ сервера и\u00a0оставляет копию в\u00a0~/.agentsync/store.\n" +
				"Публикацию эта команда не\u00a0создаёт.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync record"},
		},
		{
			name:    "store",
			summary: "Читает снимки ~/.agents из\u00a0хранилища",
			usage:   "agentsync store [номер]",
			about:   "Без номера команда печатает список снимков ~/.agents. С\u00a0номером печатает один снимок и\u00a0сохраняет его копию локально.",
			envs:    []envLine{root, server, token, cookie},
			examples: []string{
				"agentsync store",
				"agentsync store 2",
			},
		},
		{
			name:    "revert",
			summary: "Возвращает последний снимок на этой машине",
			usage:   "agentsync revert <ии-агент|agents>",
			about: "Команда берёт последний локальный снимок этого аккаунта для agents или указанного ИИ-агента.\n" +
				"После отката снимок из\u00a0локальной цепочки уходит.\n" +
				"Публикации на сервере не\u00a0меняются.",
			envs:     []envLine{root, account},
			examples: []string{"agentsync revert agents", "agentsync revert claude"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента или agents.",
		},
		{
			name:    "export",
			summary: "Переносит переносимую настройку ИИ-агентов в\u00a0~/.agents",
			usage:   "agentsync export [--only виды] [--skills] [--agents] [--commands] [--instructions] [--mcp] [--plugins] [--hooks] [ии-агент…]",
			about: "Без аргументов команда обходит найденных на\u00a0машине ИИ-агентов из\u00a0каталога.\n" +
				"С\u00a0именами берёт только перечисленных ИИ-агентов.\n" +
				"Флаг --only (через запятую или несколько раз) и\u00a0флаги видов (--skills, --agents, --commands, --instructions, --mcp, --plugins, --hooks)\n" +
				"ограничивают перенос выбранными видами.\n" +
				"Навыки, субагенты, пользовательские команды, файл инструкций, MCP-серверы, плагины и\u00a0хуки\n" +
				"переводятся в\u00a0формат agents и\u00a0кладутся в\u00a0~/.agents. Файлы самих ИИ-агентов остаются на\u00a0месте.\n" +
				"Машинные MCP и\u00a0машинные хуки остаются у\u00a0ИИ-агента.\n" +
				"Если имя элемента уже занято в\u00a0~/.agents, элемент в\u00a0~/.agents остаётся, вывод называет пропущенное.\n" +
				"Перед записью ~/.agents сохраняется в\u00a0снимок: откат\u00a0— agentsync revert agents.\n" +
				"Чтобы раздать элементы из\u00a0~/.agents ИИ-агенту в\u00a0его формате, запустите agentsync <ии-агент> on.\n" +
				"Аккаунт и\u00a0сервер не\u00a0нужны.",
			envs:     []envLine{root, account},
			examples: []string{"agentsync export", "agentsync export --only skills,mcp", "agentsync export claude --plugins --hooks"},
		},
		{
			name:    "on",
			summary: "Подключает ИИ-агента к\u00a0~/.agents: элементы agents видны ему ссылками",
			usage:   "agentsync <ии-агент> on",
			about: "Каждый внешний навык из\u00a0~/.agents/skills появляется в\u00a0глобальном каталоге навыков ИИ-агента ссылкой.\n" +
				"Субагенты (~/.agents/agents), пользовательские команды (~/.agents/commands) и\u00a0файл инструкций ~/.agents/AGENTS.md\n" +
				"появляются там, где их читает ИИ-агента: у\u00a0Claude Code файл инструкций называется CLAUDE.md, у\u00a0Gemini CLI GEMINI.md.\n" +
				"MCP-серверы из\u00a0~/.agents/mcp.json вписываются в\u00a0файл настроек ИИ-агента в\u00a0его формате: Claude Code ~/.claude.json,\n" +
				"Codex ~/.codex/config.toml, Gemini CLI ~/.gemini/settings.json. Чужие записи с\u00a0другими именами остаются.\n" +
				"Одноимённая чужая запись заменяется, команда off её возвращает. Codex не\u00a0раскрывает ${NAME}:\n" +
				"значение берётся из\u00a0окружения или из\u00a0~/.agents/.env. Без значения переменная вписывается как есть, вывод её называет.\n" +
				"Плагины из\u00a0~/.agents/plugins.json (enabledPlugins, extraKnownMarketplaces) вписываются в\u00a0settings.json Claude Code\n" +
				"по\u00a0тому же учёту своих записей. Сам плагин скачивает ИИ-агент.\n" +
				"Хуки из\u00a0~/.agents/hooks.json (события Claude Code) вписываются в\u00a0settings.json Claude Code как есть,\n" +
				"а\u00a0для Gemini CLI переводятся в\u00a0его события. Хук события, которого у\u00a0ИИ-агента нет, пропускается.\n" +
				"Хук с\u00a0абсолютным путём остаётся на машине: в\u00a0снимок он не входит, применение его не убирает.\n" +
				"Вид, который ИИ-агента не\u00a0понимает, пропускается, и\u00a0вывод его называет.\n" +
				"Правка в\u00a0agents сразу видна ИИ-агенту. Повторная команда убирает ссылку на элемент, которого в\u00a0agents больше нет.\n" +
				"Если имя занято собственной папкой или файлом ИИ-агента, команда сначала создаёт его снимок, затем ставит ссылку.\n" +
				"Аккаунт, подтверждение машины и\u00a0сервер не нужны.",
			envs:     []envLine{root},
			examples: []string{"agentsync claude on", "agentsync cursor on"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "off",
			summary: "Отключает ИИ-агента от\u00a0agents, agents остаётся",
			usage:   "agentsync <ии-агент> off",
			about: "Ссылки на agents уходят из\u00a0всех мест ИИ-агента: навыков, субагентов, команд и\u00a0файла инструкций, в\u00a0том числе ссылки на удалённые элементы.\n" +
				"Из\u00a0файлов настроек уходят только записи MCP, плагинов и\u00a0хуков, которые вписала команда on. Заменённые чужие записи возвращаются.\n" +
				"Собственные файлы и\u00a0папки ИИ-агента и\u00a0сам agents остаются. Аккаунт и\u00a0сервер не нужны.\n" +
				"Собственный файл, который on убрал в\u00a0снимок ради ссылки, возвращает agentsync revert <ии-агент>.",
			envs:     []envLine{root},
			examples: []string{"agentsync claude off"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "apply",
			summary: "Применяет публикацию agents к\u00a0~/.agents",
			usage:   "agentsync apply <автор> [--version <номер> <путь>…]",
			about: "Без --version публикация agents автора точно заменяет ~/.agents, подключённые ИИ-агенты подключаются заново.\n" +
				"Перед заменой ~/.agents записывается в\u00a0снимок: откат\u00a0— agentsync revert agents.\n" +
				"Файл ~/.agents/.env, машинный MCP и\u00a0машинные хуки остаются на\u00a0месте.\n" +
				"Своя публикация требует cookie сессии этого аккаунта. Чужую можно применить без входа на\u00a0сайт.\n" +
				"С\u00a0--version и\u00a0путями команда берёт названную версию и\u00a0кладёт в\u00a0~/.agents только названные пути.\n" +
				"Путь\u00a0— файл или папка версии, как на\u00a0странице версии. Папка с\u00a0SKILL.md ложится в\u00a0~/.agents/skills/<имя>.\n" +
				"Субагенты, команды, файл инструкций и\u00a0записи MCP, плагинов и\u00a0хуков ложатся на\u00a0свои места в\u00a0agents.\n" +
				"Одноимённый элемент заменяется целиком, остальные остаются. Файл, который ничего не\u00a0даёт agents, пропускается, вывод его называет.\n" +
				"Открытую версию команда берёт без входа. Скрытую\u00a0— с\u00a0сессией владельца или токеном его машины.\n" +
				"Пароль в\u00a0команду не\u00a0входит.",
			envs: []envLine{root, server, token, cookie},
			examples: []string{
				"agentsync apply Author",
				"agentsync apply Author --version 4 skills/ru-text agents/reviewer.md AGENTS.md",
			},
			minArgs: 1,
			missing: "Назовите автора.",
		},
		{
			name:     "version",
			summary:  "Печатает версию этой команды",
			usage:    "agentsync version",
			about:    "Номер вшивается при сборке. Сборка без номера печатает dev.",
			examples: []string{"agentsync version"},
		},
		{
			name:    "update",
			summary: "Обновляет команду agentsync до опубликованной сборки",
			usage:   "agentsync update [--check]",
			about: "Команда скачивает бинарник с того же адреса, что и установка, и заменяет текущий файл.\n" +
				"Сервер без описания версии тоже подходит: команда сравнивает содержимое файла.\n" +
				"Флаг --check только сообщает, есть ли другая сборка, и файл не меняет.\n" +
				"Копия в ~/.agentsync проверяет обновление сама, не чаще раза в сутки, перед другими командами.\n" +
				"AGENTSYNC_AUTO_UPDATE=0 выключает эту проверку. Команда agentsync update при этом работает.\n" +
				"Если локальный номер новее опубликованного, автоматическая замена не выполняется.\n" +
				"Пароль в команду не входит.",
			envs: []envLine{
				{name: "AGENTSYNC_SERVER", text: "Адрес API"},
				{name: "AGENTSYNC_AUTO_UPDATE", text: "0 выключает автоматическую проверку"},
			},
			examples: []string{
				"agentsync update",
				"agentsync update --check",
			},
		},
		{
			name:    "help",
			summary: "Показывает справку",
			usage:   "agentsync help [команда]",
			about:   "Без аргумента показывает общий список. С\u00a0именем команды показывает справку только по ней.",
			examples: []string{
				"agentsync help",
				"agentsync help apply",
			},
		},
	}
}

func commandByName(name string) (spec, bool) {
	for _, item := range commands() {
		if item.name == name {
			return item, true
		}
	}
	return spec{}, false
}

func rootHelp() string {
	var b strings.Builder
	b.WriteString("Локальный агент AgentSync переносит переносимую настройку ~/.agents.\n\n")
	b.WriteString("Использование:\n")
	b.WriteString("  agentsync <команда> [аргументы]\n\n")
	b.WriteString("Команды:\n")
	rows := make([][2]string, 0, len(commands()))
	for _, item := range commands() {
		rows = append(rows, [2]string{item.name, item.summary})
	}
	b.WriteString(formatRows(rows))
	b.WriteString("\nПеременные окружения:\n")
	b.WriteString(formatRows([][2]string{
		{"AGENTSYNC_ROOT", "Каталог с\u00a0домашними папками ИИ-агентов. По умолчанию домашний каталог"},
		{"AGENTSYNC_SERVER", "Адрес API. По умолчанию https://agentsync.zwarder.ru/api"},
		{"AGENTSYNC_TOKEN", "Токен аккаунта. Файл в\u00a0домашнем каталоге: ~/.agentsync/token"},
		{"AGENTSYNC_ACCOUNT", "Имя аккаунта, чья цепочка лежит на машине"},
		{"AGENTSYNC_COOKIE", "Cookie сессии. Файл в\u00a0домашнем каталоге: ~/.agentsync/cookie"},
		{"AGENTSYNC_AUTO_UPDATE", "0 выключает автоматическое обновление копии в ~/.agentsync"},
	}))
	b.WriteString("\nПримеры:\n")
	b.WriteString(formatExamples([]string{
		"agentsync login",
		"agentsync export",
		"agentsync claude on",
		"agentsync push",
		"agentsync store 2",
		"agentsync apply Author",
		"agentsync apply Author --version 4 skills/ru-text",
		"agentsync version",
		"agentsync update",
		"agentsync help apply",
	}))
	b.WriteString("\nЧтобы открыть справку по команде, запустите agentsync help <команда>.\n")
	return b.String()
}

func (item spec) help() string {
	var b strings.Builder
	b.WriteString(item.summary)
	b.WriteString("\n\nИспользование:\n  ")
	b.WriteString(item.usage)
	b.WriteString("\n\n")
	b.WriteString(item.about)
	b.WriteString("\n")
	if len(item.envs) > 0 {
		b.WriteString("\nПеременные окружения:\n")
		rows := make([][2]string, 0, len(item.envs))
		for _, line := range item.envs {
			rows = append(rows, [2]string{line.name, line.text})
		}
		b.WriteString(formatRows(rows))
	}
	if len(item.examples) > 0 {
		b.WriteString("\nПримеры:\n")
		b.WriteString(formatExamples(item.examples))
	}
	return b.String()
}

func formatRows(rows [][2]string) string {
	width := 0
	for _, row := range rows {
		if len(row[0]) > width {
			width = len(row[0])
		}
	}
	var b strings.Builder
	for _, row := range rows {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, row[0], row[1])
	}
	return b.String()
}

func formatExamples(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		fmt.Fprintf(&b, "  %s\n", line)
	}
	return b.String()
}
