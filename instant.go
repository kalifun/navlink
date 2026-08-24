package navlink

import (
	"context"
	"strings"

	vda5050 "github.com/kalifun/vda5050-types-go"
	"github.com/kalifun/vda5050-types-go/instant_actions"

	"github.com/kalifun/navlink/internal/gerrors"
)

// InitPositionParams is the VDA5050 initPosition parameter set (2.0 / 2.1).
// Keys on the wire are x, y, theta, mapId, lastNodeId — there is no lastNodeSequenceId.
type InitPositionParams struct {
	X          float64
	Y          float64
	Theta      float64
	MapID      string
	LastNodeID string
}

func (a *AGVHandle) publishStdInstant(ctx context.Context, headerID uint32, actionID, actionType string, params []vda5050.ActionParameter) (PublishResult, error) {
	if actionID == "" {
		return PublishResult{}, gerrors.NewInvalidConfigWithArgs("actionId is required")
	}
	ia := &instant_actions.InstantActions{
		Actions: []instant_actions.InstantAction{
			{
				ActionType:   actionType,
				ActionId:     actionID,
				BlockingType: vda5050.Hard,
				Parameters:   params,
			},
		},
	}
	ia.HeaderId = headerID
	return a.PublishInstantActions(ctx, ia)
}

// CancelOrder publishes a standard cancelOrder instantAction.
// actionID and headerID must be supplied by the caller (orchestration layer).
func (a *AGVHandle) CancelOrder(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionCancelOrder, nil)
}

// StartPause publishes a standard startPause instantAction.
func (a *AGVHandle) StartPause(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionStartPause, nil)
}

// StopPause publishes a standard stopPause instantAction.
func (a *AGVHandle) StopPause(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionStopPause, nil)
}

// StateRequest publishes a standard stateRequest instantAction.
func (a *AGVHandle) StateRequest(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionStateRequest, nil)
}

// FactsheetRequest publishes a standard factsheetRequest instantAction.
// This action is defined by VDA5050 2.1.0; the library does not gate it on Config.Version.
func (a *AGVHandle) FactsheetRequest(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionFactsheetRequest, nil)
}

// StartCharging publishes the official startCharging instantAction (no parameters).
// Vendor-specific charging parameters are not part of this helper; use PublishInstantActions.
func (a *AGVHandle) StartCharging(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionStartCharging, nil)
}

// StopCharging publishes the official stopCharging instantAction (no parameters).
func (a *AGVHandle) StopCharging(ctx context.Context, headerID uint32, actionID string) (PublishResult, error) {
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionStopCharging, nil)
}

// InitPosition publishes a standard initPosition instantAction.
func (a *AGVHandle) InitPosition(ctx context.Context, headerID uint32, actionID string, p InitPositionParams) (PublishResult, error) {
	if strings.TrimSpace(p.MapID) == "" {
		return PublishResult{}, gerrors.NewInvalidConfigWithArgs("mapId is required")
	}
	if strings.TrimSpace(p.LastNodeID) == "" {
		return PublishResult{}, gerrors.NewInvalidConfigWithArgs("lastNodeId is required")
	}
	params := []vda5050.ActionParameter{
		{Key: "x", Value: p.X},
		{Key: "y", Value: p.Y},
		{Key: "theta", Value: p.Theta},
		{Key: "mapId", Value: p.MapID},
		{Key: "lastNodeId", Value: p.LastNodeID},
	}
	return a.publishStdInstant(ctx, headerID, actionID, vda5050.ActionInitPosition, params)
}
