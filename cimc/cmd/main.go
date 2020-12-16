package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anuvu/axepect/cimc"
	"github.com/anuvu/axepect/loginshell"
	"github.com/apex/log"
	"github.com/urfave/cli/v2"
)

var version string

func main() {
	app := &cli.App{
		Name:    "cimc",
		Version: version,
		Usage:   "Play around with cimc",
		Commands: []*cli.Command{
			&cli.Command{
				Name:   "demo",
				Usage:  "demo the cimc stuff",
				Action: demoMain,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "serial-login",
						Usage: "Attempt serial login over SOL with user:pass",
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}

func demoMain(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("Got %d args, expected 1 (user:pass@ip)", c.Args().Len())
	}

	userHostPass := c.Args().First()
	toks := strings.SplitN(userHostPass, "@", 2)
	host := toks[1]
	toks = strings.SplitN(toks[0], ":", 2)
	user := toks[0]
	pass := toks[1]

	loginCreds := c.String("serial-login")

	cs, err := cimc.NewSession(host+":22", user, pass)
	if err != nil {
		log.Fatalf("failed new session: %v", err)
	}

	fmt.Printf("Connected to cimc %s\n", cs)

	for _, cmd := range []string{"show sol", "show http", "/bios/show"} {
		fmt.Printf("> %s\n", cmd)
		ret, err := cs.SendCmd(cmd)
		if err != nil {
			log.Fatalf("Failed: %v\n", err)
		}
		fmt.Printf("%s", ret)
	}

	if loginCreds != "" {
		toks := strings.SplitN(loginCreds, ":", 2)
		loginUser := toks[0]
		loginPass := toks[1]

		exp, err := cs.ExpectHostConsole()
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		log.Info(fmt.Sprintf("Connected to host console, attempting login as '%s'", loginUser))
		exp.Send("\n\n")

		rshell, err := loginshell.Login(exp, loginUser, loginPass)
		if err != nil {
			log.Fatalf("Failed to login: %v\n", err)
		}

		cmdR := rshell.Run("acs debug-token\n")
		if cmdR.RC != 0 {
			log.Fatalf("Failed to send command: \n%s\n", cmdR.String())
		}

		if err := rshell.Logout(); err != nil {
			log.Fatalf("Failed to logout: %v\n", err)
		}

		if err := cs.EndHostConsole(); err != nil {
			log.Fatalf("Failed to exit host console\n")
		}

	}

	if out, err := cs.SendCmd("/show version"); err != nil {
		log.Fatalf("Failed to show version: %v", err)
	} else {
		fmt.Printf("Version seems to be:%s\n", out)
	}

	if err := cs.Close(); err != nil {
		log.Fatalf("Failed to close session: %v\n", err)
	}

	fmt.Printf("All done\n")

	return nil
}
