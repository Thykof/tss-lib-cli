package main

import (
	"log"
	"os"
	"strconv"

	"github.com/Thykof/tss-lib-cli/internal/generate"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "tss-lib-cli",
		Usage: "CLI tool for tss-lib",
		Commands: []*cli.Command{
			{
				Name:	"generate",
				Aliases: []string{"g"},
				Usage:   "generate a new key pair",
				Action: func(cCtx *cli.Context) error {
					n, err := strconv.Atoi(cCtx.Args().First())
					if err != nil {
						return err
					}
					t, err := strconv.Atoi(cCtx.Args().Get(1))
					if err != nil {
						return err
					}
					err = generate.Generate(n, t)
					return err
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}