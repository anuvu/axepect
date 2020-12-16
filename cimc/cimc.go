package cimc

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	goexpect "github.com/google/goexpect"
	"github.com/google/goterm/term"
	"golang.org/x/crypto/ssh"
)

const (
	timeout = 60 * time.Second
	ctrlX   = "\x18"
	ctrlM   = "\x0D"
)

// Session - object holding info for the cimc session.
type Session struct {
	sshClient *ssh.Client
	exp       *goexpect.GExpect
	desc      string
	promptRe  *regexp.Regexp
}

// NewSession - return a Session, logging in with password and user@addr
//   For example NewSession("10.0.0.1", "admin", "password")
func NewSession(addr, user, pass string) (Session, error) {
	sess := Session{}
	fmt.Printf("Connecting to %s@%s\n", user, addr)
	sshClt, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})

	sess.desc = user + "@" + addr
	sess.sshClient = sshClt

	if err != nil {
		return sess, err
	}

	// we can't simply use the SpawnSSH for connecting to a cimc because
	// the cimc basically refuses to send content if ssh.ECHO is set to 0
	// Presumably that is to attempt to force people to use it interactively.
	tios := term.Termios{}
	tios.Raw()
	tios.Wz.WsCol, tios.Wz.WsRow = 0, 32768
	tios.Lflag |= term.ECHO

	opts := []goexpect.Option{}
	// To debug, just add options
	// opts = []goexpect.Option{goexpect.Verbose(true), goexpect.VerboseWriter(os.Stderr)}
	e, _, err := goexpect.SpawnSSHPTY(sshClt, timeout, tios, opts...)
	if err != nil {
		return sess, err
	}

	sess.exp = e

	// on connect we expect to see 'serial#'
	// later prompt is :
	//    <serial> path #
	// or, if path is "top"
	//    <serial>#
	promptRe := regexp.MustCompile(`([-0-9a-zA-Z]*)# `)
	_, subs, err := e.Expect(promptRe, timeout)

	sess.desc = fmt.Sprintf("%s@%s [%s]", user, addr, subs[1])
	sess.promptRe = regexp.MustCompile(
		`(` + regexp.QuoteMeta(subs[1]) + `)([ ](/[^ ]*)[ ]){0,1}(#) `)

	return sess, nil
}

func (cs Session) String() string {
	return cs.desc
}

// Close - close the ssh session.
func (cs *Session) Close() error {
	if err := cs.exp.Close(); err != nil {
		return err
	}
	if err := cs.sshClient.Close(); err != nil {
		return err
	}

	return nil
}

// SendCmd - send a command to the cimc command line interface.  Return its response.
func (cs *Session) SendCmd(msg string) (string, error) {
	fields := strings.Fields(msg)
	if strings.HasPrefix(msg, "/") {
		toks := strings.Split(fields[0], "/")

		// support SendCmd("/bios/memory/show detail")
		cmd := toks[len(toks)-1]
		scope := strings.Join(toks[1:len(toks)-1], "/")

		_, err := cs.SendCmd("top")
		if err != nil {
			return "", err
		}

		_, err = cs.SendCmd("scope " + scope)
		if err != nil {
			return "", err
		}

		msg = cmd
		if len(fields) != 1 {
			msg += " " + strings.Join(fields[1:], " ")
		}
	}

	send := msg + " | no-more"
	// do not add 'no-more' to top, scope, or any command with a |
	if fields[0] == "top" || fields[0] == "scope" || strings.ContainsAny(msg, "|") {
		send = msg
	}

	if err := cs.exp.Send(send + "\n"); err != nil {
		return "", err
	}

	// data has
	//  * the command we sent (due to ECHO)
	//  * multi line response
	//  * prompt line
	data, _, err := cs.exp.Expect(cs.promptRe, timeout)
	if err != nil {
		return "", err
	}

	fulldata := strings.Replace(data, ctrlM, "", -1)
	lines := strings.Split(fulldata, "\n")

	if len(lines) < 2 {
		return "", fmt.Errorf("Failed to parse response from '%s': %s", send, data)
	}

	promptLine := strings.TrimSpace(lines[len(lines)-1])

	// sometimes we get multiple prompt lines in response.
	dataLines := []string{}
	for _, line := range lines[1 : len(lines)-1] {
		if strings.TrimSpace(line) == promptLine {
			continue
		}
		dataLines = append(dataLines, line)
	}
	response := strings.Join(dataLines, "\n")

	return response + "\n", nil
}

// ExpectHostConsole - return a expect.GExpect that is hooked up to the host's console.
// as you would get if you typed 'connect host'
func (cs *Session) ExpectHostConsole() (*goexpect.GExpect, error) {
	exp := cs.exp
	if err := exp.Send("connect host\n"); err != nil {
		return nil, err
	}
	_, _, err := exp.Expect(regexp.MustCompile(regexp.QuoteMeta("Press Ctrl+x to Exit the session")), timeout)
	if err != nil {
		return nil, err
	}

	return exp, err
}

// EndHostConsole - exit from the host console, back to the cimc shell.
func (cs *Session) EndHostConsole() error {
	if _, err := cs.SendCmd(ctrlX); err != nil {
		return err
	}
	return nil
}
