package cimc

import "context"

type PowerState int

const (
	Unknown PowerState = iota
	Off
	On
)

type CIMCSession interface {
	PowerOn(context.Context) error
	PowerOff(context.Context) error
	PowerCycle(context.Context) error
	GetPowerState(context.Context) (PowerState, error)
}
