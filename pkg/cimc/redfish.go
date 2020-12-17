package cimc

import (
	"context"
	"fmt"
	"strconv"
)

// RedfishEnable - Turn on redfish api.
func (cs *Session) RedfishEnable(ctx context.Context) error {
	return setRedfish(ctx, cs, true)
}

// RedfishDisable - Turn off redfish api
func (cs *Session) RedfishDisable(ctx context.Context) error {
	return setRedfish(ctx, cs, false)
}

// RedfishInfo - query state of redfish.
func (cs *Session) RedfishInfo(ctx context.Context) (bool, int, int, error) {
	return getRedfish(ctx, cs)
}

func getRedfish(ctx context.Context, cs *Session) (bool, int, int, error) {
	var enabled, ok bool
	var nactive, nmax int

	resp, err := cs.SendCmd(ctx, "/redfish/show detail")
	if err != nil {
		return enabled, nactive, nmax, err
	}

	deets := parseDetail(resp)

	val := deets["Enabled"]
	if val == "no" {
		enabled = false
	} else if val == "yes" {
		enabled = true
	} else {
		return enabled, nactive, nmax, fmt.Errorf("Unknown redfish 'Enabled' setting: '%s'", val)
	}

	val = deets["Active Sessions"]
	if val == "" {
		return enabled, nactive, nmax, fmt.Errorf("Empty 'Active Sessions' setting: '%s'", val)
	}

	nactive, err = strconv.Atoi(val)
	if err != nil {
		return enabled, nactive, nmax, fmt.Errorf("Failed to parse 'Active Sessions' setting: '%s'", val)
	}

	val, ok = deets["Max Sessions"]
	if !ok {
		return enabled, nactive, nmax, fmt.Errorf("No 'Max Sessions' setting: '%s'", val)
	}

	nmax, err = strconv.Atoi(val)
	if err != nil {
		return enabled, nactive, nmax, fmt.Errorf("Failed to parse 'Max Sessions' setting: '%s'", val)
	}

	return enabled, nactive, nmax, nil
}

func setRedfish(ctx context.Context, cs *Session, desired bool) error {
	val := "no"
	if desired {
		val = "yes"
	}

	enabled, _, _, err := getRedfish(ctx, cs)
	if err != nil {
		return err
	}

	if enabled == desired {
		return nil
	}

	_, err = cs.SendCmd(ctx, "set enabled "+val)
	if err != nil {
		return err
	}

	if _, err := cs.SendCmd(ctx, "commit"); err != nil {
		return err
	}

	enabled, _, _, err = getRedfish(ctx, cs)
	if err != nil {
		return fmt.Errorf("Failed to verify redfish status after commit: %v", err)
	}

	if enabled != desired {
		return fmt.Errorf("failed to set redfish enabled=%t", desired)
	}

	return nil
}
