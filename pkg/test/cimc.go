package test

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/phayes/freeport"
)

const Prompt = "CONSOLE"

func NewMockServer() (int, error) {
	port, err := freeport.GetFreePort()
	if err != nil {
		return -1, err
	}

	ssh.Handle(func(s ssh.Session) {
		io.WriteString(s, fmt.Sprintf("%s# \n", Prompt))
		cmd := make([]byte, 1024)
		for {
			n, err := s.Read(cmd)
			if err != nil {
				s.Exit(1)
				return
			}
			str := strings.TrimSpace(string(cmd[0:n]))
			log.Printf("str=%#v n=%#v err=%#v\n", str, n, err)
			switch str {
			case "top":
				io.WriteString(s, fmt.Sprintf("%s# \n", Prompt))
			case "scope chassis":
				io.WriteString(s, fmt.Sprintf("%s /chassis # \n", Prompt))
			case "show detail | no-more":
				io.WriteString(s, "\nChassis:\n Power: on\n")
				io.WriteString(s, fmt.Sprintf("%s# \n", Prompt))
			case "power on | no-more":
				io.WriteString(s, fmt.Sprintf("%s# \n", Prompt))
			case "power off | no-more":
				io.WriteString(s, fmt.Sprintf("%s# \n", Prompt))
			default:
				log.Printf("str=%#v\n", str)
			}
		}
	})

	// start the server in a different goroutine
	go func() {
		log.Printf("ssh server listening on port %d\n", port)
		log.Fatal(ssh.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()

	return port, err
}
