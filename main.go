package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "0.0.0.0"
	port = 13337
)

//go:embed asset/cv.md
var cv []byte

func curriculumMiddleware() wish.Middleware {
	return func(h ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			wish.Printf(s, "::: welcome to the matrix %s (%s)\n", s.User(), s.RemoteAddr())

			cmd := s.Command()
			if args := len(cmd); args == 0 {
				render, _ := glamour.RenderBytes(cv, "dark")
				wish.Printf(s, "%s", render)
			} else if args == 1 && cmd[0] == "status" {
				wish.Printf(s, "status")
			} else {
				wish.Errorf(s, "unknown command %s", cmd[0:10])
			}
			h(s)
		}
	}
}

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithIdleTimeout(60*time.Second),
		wish.WithMaxTimeout(60*time.Second),
		wish.WithMiddleware(
			curriculumMiddleware(),
			logging.Middleware(),
		),
	)

	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("starting ssh server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("stopping ssh server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}
