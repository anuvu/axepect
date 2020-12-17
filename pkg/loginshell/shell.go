package loginshell

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	axepect "github.com/anuvu/axepect/pkg/expect"
	"github.com/apex/log"
	goexpect "github.com/google/goexpect"
	"github.com/pkg/errors"
)

const defaultTimeout = time.Duration(30 * time.Second)
const ctrlM = "\x0D"

var exitCodeMatch = regexp.MustCompile("\n exitcode=([0-9]+) [$] ")

// CmdRet - a command execution result.
type CmdRet struct {
	Cmd     string
	Output  string
	RC      int
	Elapsed time.Duration
}

func (c *CmdRet) String() string {
	return c.StringIndent("")
}

// StringIndent - get a cmdRet String with indentation.
func (c *CmdRet) StringIndent(indent string) string {
	return fmt.Sprintf("\n%sCommand: %s\n%src: %d\n%selapsed: %0.3f\n%soutput: %s\n",
		indent, c.Cmd, indent, c.RC, indent, c.Elapsed.Seconds(), indent,
		strings.ReplaceAll(c.Output, "\n", "\n"+indent+"  "))
}

func (c *CmdRet) Error() error {
	return errors.New(c.String())
}

// ErrorRC - return a CmdRet mentioning expected rc and actual.
func (c *CmdRet) ErrorRC(rc int) error {
	return fmt.Errorf("Error: Expected RC = %d found %d.\n%s", rc, c.RC, c.String())
}

// Matches - return a regexp.MatchString on c.Output
func (c *CmdRet) Matches(match string) bool {
	matched, err := regexp.MatchString(match, c.Output)
	if err != nil {
		log.Fatalf("error in match (bad regex?): %s", match, err)
	}
	return matched
}

// AssertMatches - regexp 'match' is not found in string 'msg'
func (c *CmdRet) AssertMatches(match, msg string) {
	if !c.Matches(match) {
		log.Fatalf("%s: cmd output did not match '%s': %s\n", msg, match, c.String())
	}
}

// Shell - A login shell, set to execute comands.
type Shell struct {
	exp *goexpect.GExpect
}

// Login - Login at a 'login:' prompt with user and password.
func Login(c *goexpect.GExpect, user string, password string) (*Shell, error) {
	l := &Shell{exp: c}

	err := c.Send("\n")
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to send a newline")
	}

	buffer, _, err := axepect.ExpectWithin(c, defaultTimeout, regexp.MustCompile("login:"))
	if err != nil {
		return nil, errors.Wrapf(err, "error waiting for login prompt, got text: %s", buffer)
	}

	if err := doPassword(c, user, password); err != nil {
		return nil, err
	}

	return l, nil
}

// Poweroff - just power off the machine and wait for the final "Power down"
// message.  This is its own helper because you could not use RunBash for
// the obvious reason.
func (l *Shell) Poweroff() error {
	// Let's give those annoying systemd stop jobs > 90s to finish up
	log.Infof("Sending poweroff")
	err := l.exp.Send("poweroff\n")
	if err != nil {
		return errors.Wrapf(err, "Failed sending poweroff command")
	}
	_, _, err = axepect.ExpectWithin(l.exp, 200*time.Second, regexp.MustCompile("Power down"))
	if err != nil {
		return errors.Wrapf(err, "Error waiting for power down")
	}
	return nil
}

// Reboot - send a reboot command.
func (l *Shell) Reboot() error {
	log.Infof("rebooting")
	err := l.exp.Send("reboot\n")
	if err != nil {
		return errors.Wrapf(err, "Error sending reboot command")
	}
	return nil
}

// Logout - log the current login out.
func (l *Shell) Logout() error {
	err := l.exp.Send("logout\n")
	if err != nil {
		return errors.Wrapf(err, "Failed sending logout command")
	}
	_, _, err = axepect.ExpectWithin(l.exp, 30*time.Second, regexp.MustCompile("ogin:"))
	if err != nil {
		return errors.Wrapf(err, "Error waiting for login prompt after logout")
	}
	return nil
}

// WaitForBooted - after system is logged in, wait until systemd says it is ready.
func (l *Shell) WaitForBooted() error {
	for {
		cmdR := l.Run("systemctl is-system-running")
		if cmdR.RC != 0 && cmdR.RC != 1 {
			return cmdR.Error()
		}
		state := strings.TrimSpace(cmdR.Output)
		if state == "initializing" || state == "starting" {
			time.Sleep(time.Second)
		} else if state == "degraded" {
			l.Run("systemctl status --no-pager --full --state=failed '*'")
			return fmt.Errorf("system boot is degraded")
		} else if state == "running" {
			log.Infof("System is running.")
			return nil
		} else {
			return fmt.Errorf("system boot is unexpected state '%s'", state)
		}
	}
}

func doPassword(c *goexpect.GExpect, user string, password string) error {
	c.Send(user + "\n")
	buffer, _, err := axepect.ExpectWithin(c, defaultTimeout, regexp.MustCompile("Password:"))
	if err != nil {
		return errors.Wrapf(err, "error waiting for password prompt, got text %s", buffer)
	}

	c.Send(password + "\n")
	buffer, _, err = axepect.ExpectWithin(c, defaultTimeout,
		regexp.MustCompile(`(bash-4[.0-9]*#|\[root@[-a-z]* ~\]#)`))
	if err != nil {
		return errors.Wrapf(err, "error waiting for bash prompt, got text %s", buffer)
	}

	c.Send("export SYSTEMD_COLORS=0\n")
	c.Send("stty -echo\n")
	c.Send("PS1='\n exitcode=$? $ '\n")
	buffer, _, err = axepect.ExpectWithin(c, defaultTimeout, exitCodeMatch)
	if err != nil {
		return errors.Wrapf(err, "error setting PS1 for bash, got %s", buffer)
	}

	return nil
}

// CheckRCTimeout - execute cmd, return a CmdRet, and error if rc != cmdR.RC
func (l *Shell) CheckRCTimeout(cmd string, rc int, timeout time.Duration) (CmdRet, error) {
	cmdR := l.RunTimeout(cmd, timeout)
	if cmdR.RC != rc {
		return cmdR, cmdR.ErrorRC(rc)
	}
	return cmdR, nil
}

// CheckOnly - execute cmd and return error or nil.
func (l *Shell) CheckOnly(cmd string) error {
	cmdR := l.RunTimeout(cmd, defaultTimeout)
	if cmdR.RC != 0 {
		return cmdR.Error()
	}
	return nil
}

// Run - execute cmd, expect result in defaultTimeout
func (l *Shell) Run(cmd string) CmdRet {
	return l.RunTimeout(cmd, defaultTimeout)
}

// RunTimeout - pass 'cmd' to bash.
//   Fatalf on command execution error or timeout.
//   return error if CmdRet.RC != 0
func (l *Shell) RunTimeout(cmd string, timeout time.Duration) CmdRet {
	log.Debugf("Sending cmd %s", cmd)
	cmdR, err := l.sendCmd(cmd, timeout)
	if err != nil {
		log.Fatalf("%s", err)
	}
	log.Debugf(cmdR.StringIndent("  "))
	return cmdR
}

// sendCmd - pass "cmd" to bash. return error if
//   it does not return in 'timeout' seconds, or on harness failure.
func (l *Shell) sendCmd(cmd string, timeout time.Duration) (CmdRet, error) {
	cmdr := CmdRet{Cmd: cmd, RC: -1}

	startTime := time.Now()
	err := l.exp.Send(cmd + "\n")
	if err != nil {
		return cmdr, errors.Errorf("Failed sending command")
	}

	buffer, matches, err := axepect.ExpectWithin(l.exp, timeout, exitCodeMatch)

	cmdr.Output = buffer
	cmdr.Elapsed = time.Since(startTime)
	if err != nil {
		return cmdr, errors.Wrapf(err, "Timeout error after %s waiting on %s", timeout, cmd)
	}

	if len(matches) != 2 {
		return cmdr, errors.Errorf("bad rc matched: %v (%v) in cmd %s", cmdr.Output, matches, cmd)
	}

	promptStr := matches[0]
	rcStr := matches[1]
	rc, err := strconv.Atoi(rcStr)
	if err != nil {
		return cmdr, errors.Wrapf(err, "couldn't convert return code %s from cmd %s", rcStr, cmd)
	}
	cmdr.RC = rc

	cmdr.Output = strings.Replace(strings.TrimSpace(buffer[0:len(buffer)-len(promptStr)]), ctrlM, "", -1)
	return cmdr, nil
}
