package main

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
				"Пароль в\u00a0команду не входит.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync push Grok"},
			minArgs:  1,
			missing:  "Назовите ИИ-агента.",
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
			summary: "Кладёт выбранные навыки публикации в\u00a0каталоги приёмников",
			usage:   "agentsync skills <автор> <ии-агент> --version <номер> --global|--project --into <приёмник>… <навык>…",
			about: "Команда берёт названную версию и\u00a0копирует только названные навыки.\n" +
				"Навык\u00a0— папка с\u00a0SKILL.md. Имя можно дать коротко или путём, например skills/ru-text.\n" +
				"Навыки пишутся списком в\u00a0конце команды, каждый своим аргументом.\n" +
				"Флаг --skill тоже принимает один навык и\u00a0может повторяться.\n" +
				"Ровно один из флагов --global и\u00a0--project называет место. --into повторяется для каждого приёмника.\n" +
				"Папка навыка ложится прямо в\u00a0каталог навыков приёмника. Остальные навыки не меняются.\n" +
				"Хранилище команда не пополняет. Своя публикация требует сессии этого аккаунта. Пароль в\u00a0команду не входит.",
			envs: []envLine{root, server, token, cookie, account},
			examples: []string{
				"agentsync skills Author Grok --version 4 --global --into cursor ru-text",
				"agentsync skills Author Grok --version 4 --project --into cursor --into windsurf ru-text graphify",
			},
			minArgs: 2,
			missing: "Назовите автора и\u00a0ИИ-агента.",
		},
		{
			name:    "apply",
			summary: "Применяет публикацию к\u00a0ИИ-агенту",
			usage:   "agentsync apply <автор> <ии-агент>",
			about: "Команда заменяет переносимую настройку названного ИИ-агента файлами публикации.\n" +
				"Перед заменой текущая переносимая настройка записывается в\u00a0снимок. Откат: agentsync revert <ии-агент>.\n" +
				"Учётные данные, машинный MCP и\u00a0машинные хуки остаются на месте.\n" +
				"Своя публикация требует cookie сессии этого аккаунта. Чужую можно применить без входа на сайт.\n" +
				"Пароль в\u00a0команду не входит.",
			envs:     []envLine{root, server, token, cookie},
			examples: []string{"agentsync apply Author Grok"},
			minArgs:  2,
			missing:  "Назовите автора и\u00a0ИИ-агента.",
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
	}))
	b.WriteString("\nПримеры:\n")
	b.WriteString(formatExamples([]string{
		"agentsync login",
		"agentsync push Grok",
		"agentsync store Grok 2",
		"agentsync apply Author Grok",
		"agentsync skills Author Grok --version 4 --global --into cursor ru-text",
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
