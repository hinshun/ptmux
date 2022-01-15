package terminal

import (
	"bufio"
	"context"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/hinshun/ptmux/pkg/pubsub"
	"github.com/hinshun/vt10x"
	"github.com/rs/zerolog"
)

const (
	updateTopic = "update"
)

type Terminal struct {
	vt10x.Terminal
	ptm    *os.File
	pubsub *pubsub.Pubsub
	done   chan struct{}
}

func New(ctx context.Context) (*Terminal, error) {
	cmd := exec.CommandContext(ctx, os.Getenv("SHELL"), "-i")
	ptm, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	vt := vt10x.New(vt10x.WithWriter(ptm))
	ps := pubsub.New()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ps.Close()

		br := bufio.NewReader(ptm)
		for {
			err := vt.Parse(br)
			if err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Msg("unable to parse virtual terminal")
				break
			}

			ps.Publish(updateTopic, "")
		}
	}()

	return &Terminal{
		Terminal: vt,
		ptm:      ptm,
		pubsub:   ps,
		done:     done,
	}, nil
}

func (t *Terminal) Write(p []byte) (n int, err error) {
	return t.ptm.Write(p)
}

func (t *Terminal) Done() <-chan struct{} {
	return t.done
}

func (t *Terminal) Resize(cols, rows int) {
	vt10x.ResizePty(t.ptm, cols, rows)
	t.Terminal.Resize(cols, rows)
}

func (t *Terminal) Subscribe(id string, ch chan string) {
	t.pubsub.Subscribe(updateTopic, id, ch)
}
