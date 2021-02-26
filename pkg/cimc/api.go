package cimc

import (
	"context"

	goexpect "github.com/google/goexpect"
)

type PowerState int

const (
	Unknown PowerState = iota
	Off
	On
)

func (p PowerState) String() string {
	return []string{"Unknown", "Off", "On"}[p]
}

type CIMCSession interface {
	// PowerOn powers on the connected host
	PowerOn(context.Context) error
	// PowerOff powers off the connected host
	PowerOff(context.Context) error
	// PowerCycle powers off (if on) and then on the connected host
	PowerCycle(context.Context) error
	// GetPowerState gets the power state of the connected host
	GetPowerState(context.Context) (PowerState, error)
	// OpenConsole opens a console to the connected host
	OpenConsole(context.Context) (*goexpect.GExpect, error)
	// CloseConsole opens a console to the connected host
	CloseConsole(context.Context) error
	// SendCmd sends a command to the connected host
	SendCmd(context.Context, string) (string, error)
	// Close closes the session
	Close(context.Context) error
	// RedfishEnable turns on Redfish
	RedfishEnable(context.Context) error
	// RedfishEnable turns off Redfish
	RedfishDisable(context.Context) error
	// RedfishEnable returns info on state of Redfish
	RedfishInfo(context.Context) (bool, int, int, error)
}
