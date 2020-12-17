package cimc

import "context"

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
	PowerOn(context.Context) error
	PowerOff(context.Context) error
	PowerCycle(context.Context) error
	GetPowerState(context.Context) (PowerState, error)
}
