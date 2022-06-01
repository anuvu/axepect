package cimc

import (
	"context"
	"fmt"
	"strings"
)

// GetPowerState - return power state of system.
func (cs *Session) GetPowerState(ctx context.Context) (PowerState, error) {
	resp, err := cs.SendCmd(ctx, "/chassis/show detail")
	if err != nil {
		return Unknown, err
	}

	dets := parseDetail(resp)
	if v, ok := dets["Power"]; !ok {
		return Unknown, fmt.Errorf("did not find power state in %s", resp)
	} else if v == "on" {
		return On, nil
	} else if v == "off" {
		return Off, nil
	}

	return Unknown, fmt.Errorf("bad power state '%s'", resp)
}

// PowerOff - Turn power off, if on
func (cs *Session) PowerOff(ctx context.Context) error {
	return powerCmd(ctx, cs, "off")
}

// PowerOn - Turn power off, if off
func (cs *Session) PowerOn(ctx context.Context) error {
	return powerCmd(ctx, cs, "on")
}

// PowerCycle - Turn power off, if off and then back on.
func (cs *Session) PowerCycle(ctx context.Context) error {
	return powerCmd(ctx, cs, "cycle")
}

func powerCmd(ctx context.Context, cs *Session, cmd string) error {
	_, err := cs.SendCmd(ctx, "/chassis/power "+cmd)
	// TODO: should look at the response
	return err
}

// parse '/chassis/show detail' output.
// Expected input looks like this:
// Chassis:
//    Power: on
//    Serial Number: WZP2326007Q
//    Product Name:
//    PID : APIC-SERVER-L3
//    UUID: 13AA6335-143A-4FBE-AD2D-20487959A59B
//    Locator LED: off
//    Description:
//    Asset Tag: Unknown
func parseDetail(data string) map[string]string {
	lines := strings.Split(data, "\n")
	ret := map[string]string{}

	for _, line := range lines {
		if !strings.HasPrefix(line, " ") {
			continue
		}
		toks := strings.SplitN(line, ":", 2)
		ret[strings.TrimSpace(toks[0])] = strings.TrimSpace(toks[1])
	}
	return ret
}
