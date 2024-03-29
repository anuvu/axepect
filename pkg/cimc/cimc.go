package cimc

import (
	"context"
	"errors"
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

var noMoreCmds = []string{"commit", "top", "scope", "set", "power"}

// match the 'confirm' prompt with either y or n as default ([y|N] or [Y|n])
var confirmReStr = regexp.QuoteMeta("Do you want to continue?[") + "([yY]\\|[nN])" + regexp.QuoteMeta("]")
var confirmRe = regexp.MustCompile(confirmReStr)

// Session - object holding info for the cimc session.
type Session struct {
	sshClient         *ssh.Client
	exp               *goexpect.GExpect
	desc              string
	promptRe          *regexp.Regexp
	promptOrConfirmRe *regexp.Regexp
}

// NewSession - return a Session, logging in with password and user@addr
//   For example NewSession("10.0.0.1", "admin", "password")
func NewSession(addr, user, pass string) (CIMCSession, error) {
	return NewSessionOpts(addr, user, pass, []goexpect.Option{})
}

// NewSessionOpts - return a Session, with provided goexpect.Option list
//   For example to enable debug:
//     NewSessionOpts("10.0.0.1", "admin", "password",
//        []goexpect.Option{goexpect.Verbose(true), goexpect.VerboseWriter(os.Stderr)})
func NewSessionOpts(addr, user, pass string, opts []goexpect.Option) (CIMCSession, error) {
	sess := &Session{}
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
	if err != nil {
		return sess, err
	}

	sess.desc = fmt.Sprintf("%s@%s [%s]", user, addr, subs[1])
	promptReStr := `(` + regexp.QuoteMeta(subs[1]) + `)([ ](/[^ ]*)[ ]){0,1}([*]*)(#) `
	sess.promptRe = regexp.MustCompile(promptReStr)
	sess.promptOrConfirmRe = regexp.MustCompile(promptReStr + "|" + confirmReStr)
	return sess, nil
}

func (cs Session) String() string {
	return cs.desc
}

// Close - close the ssh session.
func (cs *Session) Close(ctx context.Context) error {
	if err := cs.exp.Close(); err != nil {
		return err
	}
	if err := cs.sshClient.Close(); err != nil {
		return err
	}

	return nil
}

// SendCmd - send a command to the cimc command line interface.  Return its response.
func (cs *Session) SendCmd(ctx context.Context, msg string) (string, error) {
	fields := strings.Fields(msg)
	cmd := fields[0]
	if strings.HasPrefix(msg, "/") {
		toks := strings.Split(fields[0], "/")

		// support SendCmd("/bios/memory/show detail")
		cmd = toks[len(toks)-1]
		scope := strings.Join(toks[1:len(toks)-1], "/")

		_, err := cs.SendCmd(ctx, "top")
		if err != nil {
			return "", err
		}

		_, err = cs.SendCmd(ctx, "scope "+scope)
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
	if strings.ContainsAny(msg, "|") {
		send = msg
	} else {
		for _, n := range noMoreCmds {
			if cmd == n {
				send = msg
				break
			}
		}
	}

	if err := cs.exp.Send(send + "\n"); err != nil {
		return "", err
	}

	// data has
	//  * the command we sent (due to ECHO)
	//  * multi line response
	//  * prompt line
	data, _, err := cs.exp.Expect(cs.promptOrConfirmRe, timeout)
	if err != nil {
		return "", err
	}
	if confirmRe.MatchString(data) {
		// Confirm prompt
		if err := cs.exp.Send("y\n"); err != nil {
			return "", fmt.Errorf("failed to send 'y' to a confirm response: %s", err)
		}
		afterConfirm, _, err := cs.exp.Expect(cs.promptRe, timeout)
		if err != nil {
			return "", fmt.Errorf("error after confirming operation: %s", err)
		}
		data += afterConfirm
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
	if len(dataLines) > 0 {
		lastLine := dataLines[len(dataLines)-1]
		if strings.HasPrefix("Error:", lastLine) {
			return response + "\n", errors.New(lastLine)
		}
	}

	return response + "\n", nil
}

// OpenConsole - return a expect.GExpect that is hooked up to the host's console.
// as you would get if you typed 'connect host'
func (cs *Session) OpenConsole(ctx context.Context) (*goexpect.GExpect, error) {
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

// CloseConsole - exit from the host console, back to the cimc shell.
func (cs *Session) CloseConsole(ctx context.Context) error {
	if _, err := cs.SendCmd(ctx, ctrlX); err != nil {
		return err
	}
	return nil
}
