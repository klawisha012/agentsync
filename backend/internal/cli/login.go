package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klawisha012/agentsync/internal/agent"
	"golang.org/x/term"
)

func login(ctx context.Context, server, email string, stdout io.Writer) error {
	email = strings.TrimSpace(email)
	if email == "" {
		fmt.Fprint(os.Stderr, "Почта: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		email = strings.TrimSpace(line)
	}
	password, err := readPassword()
	if err != nil {
		return err
	}
	name, err := agent.Login(ctx, server, email, password)
	if err != nil {
		return err
	}
	if name == "" {
		fmt.Fprintln(stdout, "Вход выполнен.")
		return nil
	}
	fmt.Fprintf(stdout, "Вход выполнен: %s.\n", name)
	return nil
}

func readPassword() (string, error) {
	if value := strings.TrimSpace(os.Getenv("AGENTSYNC_PASSWORD")); value != "" {
		return value, nil
	}
	fmt.Fprint(os.Stderr, "Пароль: ")
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
