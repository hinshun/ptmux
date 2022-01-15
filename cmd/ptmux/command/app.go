package command

import (
	cli "github.com/urfave/cli/v2"
)

func App() *cli.App {
	app := cli.NewApp()
	app.Name = "ptmux"
	app.Usage = "pair programming via terminal p2p sharing"
	app.Commands = []*cli.Command{
		hostCommand,
		joinCommand,
	}
	return app
}
