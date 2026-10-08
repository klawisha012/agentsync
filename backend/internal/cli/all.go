package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/klawisha012/agentsync/internal/agent"
)

func pushAll(ctx context.Context, server, root, token, cookie string, stdout io.Writer) error {
	names, err := agent.Found(root)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("На компьютере нет ИИ-агентов.")
	}
	failed := 0
	for _, name := range names {
		if err := agent.Push(ctx, server, root, name, token, cookie); err != nil {
			failed++
			fmt.Fprintf(stdout, "%s: %s\n", name, err.Error())
			continue
		}
		fmt.Fprintf(stdout, "Опубликован %s.\n", name)
	}
	if failed > 0 {
		return errors.New("Не все ИИ-агенты опубликованы.")
	}
	return nil
}
