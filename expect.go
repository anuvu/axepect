package axepect

import (
	"fmt"
	"regexp"
	"time"

	goexpect "github.com/google/goexpect"
)

// ExpectWithin - Call GExpect.Expect, but timeout in provided time.
func ExpectWithin(c *goexpect.GExpect, timeout time.Duration, re *regexp.Regexp) (string, []string, error) {
	myChan := make(chan struct {
		buffer  string
		matches []string
		err     error
	}, 1)
	go func() {
		buffer, matches, err := c.Expect(re, time.Minute)
		myChan <- struct {
			buffer  string
			matches []string
			err     error
		}{buffer, matches, err}
	}()

	var buffer string
	var matches []string
	var err error

	select {
	case res := <-myChan:
		buffer = res.buffer
		matches = res.matches
		err = res.err
	case <-time.After(timeout):
		buffer = ""
		matches = []string{}
		err = fmt.Errorf("Timeout after %s", timeout)
	}

	return buffer, matches, err
}
