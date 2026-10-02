#!/bin/sh
# AgentSync. Кладёт бинарник agentsync в домашний каталог и добавляет этот каталог в PATH.
# Публикацию не применяет и ИИ-агента не загружает. Пароль в эту инструкцию не входит.
set -eu

dir="${HOME}/.agentsync"
mkdir -p "$dir"

origin="${AGENTSYNC_ORIGIN:-https://zwarder.ru/api}"
origin="${origin%/}"

os=$(uname -s)
case "$os" in
  Linux) goos=linux ;;
  Darwin) goos=darwin ;;
  *)
    echo "Эта система не поддерживается." >&2
    exit 1
    ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) goarch=amd64 ;;
  aarch64|arm64) goarch=arm64 ;;
  *)
    echo "Эта архитектура не поддерживается." >&2
    exit 1
    ;;
esac

bin="$dir/agentsync"
tmp="$dir/agentsync.tmp"
url="$origin/cli/agentsync-${goos}-${goarch}"
if ! curl -fsSL "$url" -o "$tmp"; then
  rm -f "$tmp"
  echo "Не удалось скачать agentsync." >&2
  exit 1
fi
chmod 755 "$tmp"
mv "$tmp" "$bin"

rc="$HOME/.profile"
shell_name=$(basename "${SHELL:-sh}")
case "$shell_name" in
  zsh) rc="$HOME/.zshrc" ;;
  bash) rc="$HOME/.bashrc" ;;
esac
if [ ! -f "$rc" ]; then
  : > "$rc"
fi
if ! grep -F "$dir" "$rc" >/dev/null 2>&1; then
  printf '\nexport PATH="%s:$PATH"\n' "$dir" >> "$rc"
fi
export PATH="$dir:$PATH"

echo "Команда agentsync добавлена в PATH."
echo "Откройте новое окно терминала, чтобы команда agentsync нашлась."
