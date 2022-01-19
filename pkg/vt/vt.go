package vt

import (
	"bufio"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/hinshun/ptmux/pkg/pubsub"
	"github.com/hinshun/vt10x"
)

const (
	updateTopic = "update"
)

type VT struct {
	vt10x.Terminal
	ptm    *os.File
	pubsub *pubsub.Pubsub
	done   chan struct{}
}

func New(cols, rows int) (*VT, error) {
	cmd := exec.Command(os.Getenv("SHELL"), "-i")
	ptm, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
	if err != nil {
		return nil, err
	}

	vt := vt10x.New(vt10x.WithWriter(ptm), vt10x.WithSize(cols, rows))
	ps := pubsub.New()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer ps.Close()

		br := bufio.NewReader(ptm)
		for {
			err := vt.Parse(br)
			if err != nil {
				// log.Error().Err(err).Msg("unable to parse virtual terminal")
				break
			}

			ps.Publish(updateTopic, "")
		}
	}()

	return &VT{
		Terminal: vt,
		ptm:      ptm,
		pubsub:   ps,
		done:     done,
	}, nil
}

func (vt *VT) Write(p []byte) (n int, err error) {
	return vt.ptm.Write(p)
}

func (vt *VT) Done() <-chan struct{} {
	return vt.done
}

func (vt *VT) Resize(cols, rows int) {
	vt10x.ResizePty(vt.ptm, cols, rows)
	vt.Terminal.Resize(cols, rows)
}

func (vt *VT) Subscribe(id string, ch chan string) {
	vt.pubsub.Subscribe(updateTopic, id, ch)
}
