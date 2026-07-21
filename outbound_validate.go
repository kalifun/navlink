package navlink

import (
	"errors"
	"strconv"
	"strings"

	"github.com/kalifun/vda5050-types-go/instant_actions"
	"github.com/kalifun/vda5050-types-go/order"

	"github.com/kalifun/navlink/gerrors"
)

// Outbound validation reason keys (metadata "reason" on OutboundValidationFailed).
const (
	ReasonHeaderIDZero       = "headerId_zero"
	ReasonOrderIDEmpty       = "orderId_empty"
	ReasonOrderUpdateIDZero  = "orderUpdateId_zero"
	ReasonActionIDEmpty      = "actionId_empty"
	ReasonIdentityMismatch   = "identity_mismatch"
)

// OutboundValidation configures light pre-publish checks (no ID allocation).
// Nil Config.OutboundValidation means checks are enabled with defaults.
type OutboundValidation struct {
	// Disabled turns off all outbound validation.
	Disabled bool
	// AllowZeroHeaderID permits headerId == 0 (default false).
	AllowZeroHeaderID bool
	// SkipIdentityCheck skips manufacturer/serial vs AGVHandle checks (default false).
	SkipIdentityCheck bool
}

func (c Config) outboundValidation() OutboundValidation {
	if c.OutboundValidation == nil {
		return OutboundValidation{}
	}
	return *c.OutboundValidation
}

// IsOutboundValidationFailed reports a rejected bad outbound packet (not a broker failure).
func IsOutboundValidationFailed(err error) bool {
	return errors.Is(err, gerrors.OutboundValidationFailed)
}

func outboundValidationError(reason, detail string) error {
	err := gerrors.OutboundValidationFailed.With("reason", reason)
	if detail != "" {
		err = err.With("detail", detail)
	}
	return err
}

func validateOutboundOrder(o *order.Order, mfr, serial string, opts OutboundValidation) error {
	if opts.Disabled {
		return nil
	}
	if err := validateOutboundHeader(o.HeaderId, o.Manufacturer, o.SerialNumber, mfr, serial, opts); err != nil {
		return err
	}
	if strings.TrimSpace(o.OrderId) == "" {
		return outboundValidationError(ReasonOrderIDEmpty, "")
	}
	if o.OrderUpdateId == 0 {
		return outboundValidationError(ReasonOrderUpdateIDZero, "")
	}
	return nil
}

func validateOutboundInstantActions(ia *instant_actions.InstantActions, mfr, serial string, opts OutboundValidation) error {
	if opts.Disabled {
		return nil
	}
	if err := validateOutboundHeader(ia.HeaderId, ia.Manufacturer, ia.SerialNumber, mfr, serial, opts); err != nil {
		return err
	}
	for i, act := range ia.Actions {
		if strings.TrimSpace(act.ActionId) == "" {
			return outboundValidationError(ReasonActionIDEmpty, "index="+strconv.Itoa(i))
		}
	}
	return nil
}

func validateOutboundHeader(headerID uint32, msgMfr, msgSN, handleMfr, handleSN string, opts OutboundValidation) error {
	if !opts.AllowZeroHeaderID && headerID == 0 {
		return outboundValidationError(ReasonHeaderIDZero, "")
	}
	if opts.SkipIdentityCheck {
		return nil
	}
	if msgMfr != "" && msgMfr != handleMfr {
		return outboundValidationError(ReasonIdentityMismatch, "manufacturer")
	}
	if msgSN != "" && msgSN != handleSN {
		return outboundValidationError(ReasonIdentityMismatch, "serialNumber")
	}
	return nil
}
