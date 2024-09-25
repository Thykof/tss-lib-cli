package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Thykof/tss-lib-cli/internal/participant"
	"github.com/Thykof/tss-lib-cli/internal/verifier"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "tss-lib-cli",
		Usage: "CLI tool for tss-lib",
		Commands: []*cli.Command{
			{
				Name:    "generate",
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
					err = participant.Generate(n, t)
					return err
				},
			},
			{
				Name:    "sign",
				Aliases: []string{"s"},
				Usage:   "sign a message",
				Action: func(cCtx *cli.Context) error {
					n, err := strconv.Atoi(cCtx.Args().First())
					if err != nil {
						return err
					}
					t, err := strconv.Atoi(cCtx.Args().Get(1))
					if err != nil {
						return err
					}

					msg := cCtx.Args().Get(2)

					err = participant.Sign(n, t, msg)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:    "verify",
				Aliases: []string{"v"},
				Usage:   "verify the signature",
				Action: func(cCtx *cli.Context) error {
					msg := cCtx.Args().First()

					isOk, err := verifier.Verify(msg)
					if err != nil {
						return err
					}

					if isOk {
						fmt.Println("V Signature is valid V")
					} else {
						fmt.Println("X Signature is invalid X")
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
