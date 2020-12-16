package main

import (
	"fmt"
	"log"

	"github.com/anuvu/axepect/cimc"
	"github.com/anuvu/axepect/loginshell"
)

func main() {
	cs, err := cimc.NewSession("10.99.99.99", "admin", "FIXMEPLEASE")
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

	exp, err := cs.ExpectHostConsole()
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	exp.Send("\n\n")

	rshell, err := loginshell.Login(exp, "rescue-user", "FIXMEPLEASE")
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

	if out, err := cs.SendCmd("/show version"); err != nil {
		log.Fatalf("Failed to show version: %v", err)
	} else {
		fmt.Printf("Version seems to be:%s\n", out)
	}

	if err := cs.Close(); err != nil {
		log.Fatalf("Failed to close session: %v\n", err)
	}

	fmt.Printf("All done\n")
}
