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
			summary: "Публикует переносимую настройку одного ИИ-агента",
			usage:   "agentsync push <ии-агент>",
			about: "На сервер уходят только переносимые файлы. Учётные данные в\u00a0запрос не входят.\n" +
				"Нужен вход в\u00a0аккаунт. Подтверждение почты для публикации не нужно.\n" +
				"Канон ~/.agents публикуется под именем agents, ~/.agents/.env в\u00a0снимок не входит.\n" +
				"Пароль в\u00a0команду не входит.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync push Grok", "agentsync push agents"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "all",
			summary: "Публикует всех найденных ИИ-агентов",
			usage:   "agentsync all",
			about: "Команда ищет домашние папки в\u00a0каталоге с\u00a0ИИ-агентами.\n" +
				"Папка подходит, если в\u00a0ней сразу лежат навыки, правила, конфиг или AGENTS.md.\n" +
				"Каждая такая папка публикуется отдельно, тем\u00a0же способом, что agentsync push.\n" +
				"Имя берётся из имени папки. Точка в\u00a0начале имени не входит в\u00a0имя ИИ-агента.\n" +
				"Пустая папка и\u00a0каталог .agentsync не публикуются.\n" +
				"Если одна публикация не проходит, команда продолжает остальные.\n" +
				"Нужен вход в\u00a0аккаунт. Пароль в\u00a0команду не входит.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync all"},
		},
		{
			name:    "record",
			summary: "Записывает снимок в\u00a0хранилище",
			usage:   "agentsync record <ии-агент>",
			about: "Снимок неизменяемый. Команда печатает ответ сервера и\u00a0оставляет копию в\u00a0~/.agentsync/store.\n" +
				"Публикацию эта команда не создаёт.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync record Grok"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "store",
			summary: "Читает снимки из хранилища",
			usage:   "agentsync store <ии-агент> [номер]",
			about:   "Без номера команда печатает список снимков. С\u00a0номером печатает один снимок и\u00a0сохраняет его копию локально.",
			envs:    []envLine{root, server, token, cookie},
			examples: []string{
				"agentsync store Grok",
				"agentsync store Grok 2",
			},
			minArgs: 1,
			missing: "Назовите ИИ-агента.",
		},
		{
			name:    "revert",
			summary: "Возвращает последний снимок на этой машине",
			usage:   "agentsync revert <ии-агент>",
			about: "Команда берёт последний снимок цепочки этого аккаунта и\u00a0этого ИИ-агента. После отката снимок из цепочки уходит.\n" +
				"Публикации на сервере не меняются.",
			envs:     []envLine{root, account},
			examples: []string{"agentsync revert Grok"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "skills",
			summary: "Кладёт выбранные навыки публикации в\u00a0каталоги ИИ-агентов",
			usage:   "agentsync skills <автор> <ии-агент> --version <номер> [--global|--project] [--into <ии-агент>…] <навык>…",
			about: "Команда берёт названную версию и\u00a0копирует только названные навыки.\n" +
				"Навык\u00a0— папка с\u00a0SKILL.md. Имя можно дать коротко или путём, например skills/ru-text.\n" +
				"Навыки пишутся списком в\u00a0конце команды, каждый своим аргументом.\n" +
				"Флаг --skill тоже принимает один навык и\u00a0может повторяться.\n" +
				"Без --into и\u00a0без --global или --project команда сначала спрашивает ИИ-агентов, затем видимость: текущий проект или глобально. Стрелки, поиск, пробел, Enter.\n" +
				"Ровно один из флагов --global и\u00a0--project называет место. --into повторяется для каждого ИИ-агента.\n" +
				"Папка навыка ложится прямо в\u00a0каталог навыков ИИ-агента. Остальные навыки не меняются.\n" +
				"Хранилище команда не пополняет. Вход в\u00a0аккаунт не нужен.\n" +
				"Открытую версию команда копирует без сессии и\u00a0без токена.\n" +
				"Скрытую версию команда копирует с\u00a0сессией этого аккаунта или с\u00a0токеном его машины.\n" +
				"Пароль в\u00a0команду не входит.",
			envs: []envLine{
				root,
				server,
				{name: "AGENTSYNC_TOKEN", text: "Не обязателен. Токен машины владельца открывает его скрытую версию"},
				{name: "AGENTSYNC_COOKIE", text: "Не обязательна. Сессия владельца открывает его скрытую версию"},
			},
			examples: []string{
				"agentsync skills Author Grok --version 4 ru-text",
				"agentsync skills Author Grok --version 4 --global --into cursor ru-text",
			},
			minArgs: 2,
			missing: "Назовите автора и\u00a0ИИ-агента.",
		},
		{
			name:    "on",
			summary: "Подключает ИИ-агента к\u00a0канону ~/.agents: элементы канона видны ему ссылками",
			usage:   "agentsync <ии-агент> on",
			about: "Каждый внешний навык из\u00a0~/.agents/skills появляется в\u00a0глобальном каталоге навыков ИИ-агента ссылкой.\n" +
				"Субагенты (~/.agents/agents), пользовательские команды (~/.agents/commands) и\u00a0файл инструкций ~/.agents/AGENTS.md\n" +
				"появляются там, где их читает ИИ-агент: у\u00a0Claude Code файл инструкций называется CLAUDE.md, у\u00a0Gemini CLI GEMINI.md.\n" +
				"MCP-серверы из\u00a0~/.agents/mcp.json вписываются в\u00a0файл настроек ИИ-агента в\u00a0его формате: Claude Code ~/.claude.json,\n" +
				"Codex ~/.codex/config.toml, Gemini CLI ~/.gemini/settings.json. Чужие записи с\u00a0другими именами остаются.\n" +
				"Одноимённая чужая запись заменяется, команда off её возвращает. Codex не\u00a0раскрывает ${NAME}:\n" +
				"значение берётся из\u00a0окружения или из\u00a0~/.agents/.env.\n" +
				"Плагины из\u00a0~/.agents/plugins.json (enabledPlugins, extraKnownMarketplaces) вписываются в\u00a0settings.json Claude Code\n" +
				"по\u00a0тому же учёту своих записей. Сам плагин скачивает ИИ-агент.\n" +
				"Хуки из\u00a0~/.agents/hooks.json (события Claude Code) вписываются в\u00a0settings.json Claude Code как есть,\n" +
				"а\u00a0для Gemini CLI переводятся в\u00a0его события. Хук события, которого у\u00a0ИИ-агента нет, пропускается.\n" +
				"Хук с\u00a0абсолютным путём остаётся на машине: в\u00a0снимок он не входит, применение его не убирает.\n" +
				"Вид, который ИИ-агент не\u00a0понимает, пропускается, и\u00a0вывод его называет.\n" +
				"Правка в\u00a0каноне сразу видна ИИ-агенту. Повторная команда убирает ссылку на элемент, которого в\u00a0каноне больше нет.\n" +
				"Если имя занято собственной папкой или файлом ИИ-агента, команда сначала создаёт его снимок, затем ставит ссылку.\n" +
				"Аккаунт, подтверждение машины и\u00a0сервер не нужны.",
			envs:     []envLine{root},
			examples: []string{"agentsync claude on", "agentsync cursor on"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "off",
			summary: "Отключает ИИ-агента от\u00a0канона, канон остаётся",
			usage:   "agentsync <ии-агент> off",
			about: "Ссылки на канон уходят из\u00a0всех мест ИИ-агента: навыков, субагентов, команд и\u00a0файла инструкций, в\u00a0том числе ссылки на удалённые элементы.\n" +
				"Из\u00a0файлов настроек уходят только записи MCP, плагинов и\u00a0хуков, которые вписала команда on. Заменённые чужие записи возвращаются.\n" +
				"Собственные файлы и\u00a0папки ИИ-агента и\u00a0сам канон остаются. Аккаунт и\u00a0сервер не нужны.\n" +
				"Собственный файл, который on убрал в\u00a0снимок ради ссылки, возвращает agentsync revert <ии-агент>.",
			envs:     []envLine{root},
			examples: []string{"agentsync claude off"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
		},
		{
			name:    "apply",
			summary: "Применяет публикацию к\u00a0ИИ-агенту",
			usage:   "agentsync apply <автор> <ии-агент>",
			about: "Элементы канона из\u00a0публикации (навыки, субагенты, команды, файл инструкций, MCP) ложатся в\u00a0канон ~/.agents:\n" +
				"одноимённые заменяются целиком, остальные остаются. Собственные файлы ИИ-агента ложатся в\u00a0его папку по\u00a0тому же правилу.\n" +
				"Затем ИИ-агент подключается к\u00a0канону, как после agentsync <ии-агент> on.\n" +
				"Публикация канона (agents) точно заменяет канон, подключённые ИИ-агенты подключаются заново.\n" +
				"Перед заменой канон и\u00a0переносимая настройка ИИ-агента записываются в\u00a0снимки.\n" +
				"Откат ИИ-агента: agentsync revert <ии-агент>. Откат канона: agentsync revert agents.\n" +
				"Учётные данные, машинный MCP и\u00a0машинные хуки остаются на месте.\n" +
				"Своя публикация требует cookie сессии этого аккаунта. Чужую можно применить без входа на сайт.\n" +
				"Пароль в\u00a0команду не входит.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync apply Author Grok"},
			minArgs:  2,
			missing:  "Назовите автора и\u00a0ИИ-агента.",
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
	b.WriteString("Локальный агент AgentSync переносит переносимую настройку ИИ-агента.\n\n")
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
		{"AGENTSYNC_SERVER", "Адрес API. По умолчанию https://zwarder.ru/api"},
		{"AGENTSYNC_TOKEN", "Токен аккаунта. Файл в\u00a0домашнем каталоге: ~/.agentsync/token"},
		{"AGENTSYNC_ACCOUNT", "Имя аккаунта, чья цепочка лежит на машине"},
		{"AGENTSYNC_COOKIE", "Cookie сессии. Файл в\u00a0домашнем каталоге: ~/.agentsync/cookie"},
		{"AGENTSYNC_AUTO_UPDATE", "0 выключает автоматическое обновление копии в ~/.agentsync"},
	}))
	b.WriteString("\nПримеры:\n")
	b.WriteString(formatExamples([]string{
		"agentsync login",
		"agentsync push Grok",
		"agentsync all",
		"agentsync store Grok 2",
		"agentsync apply Author Grok",
		"agentsync skills Author Grok --version 4 ru-text",
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
