package navlink

// Identity is the protocol-level AGV identity (manufacturer + serial).
type Identity struct {
	Manufacturer string
	SerialNumber string
}

// IdentityMapper maps protocol identity (mfr+sn) to a platform robot ID.
// Injected by the platform on Config; fills Envelope.RobotID on inbound paths.
// navlink never invents robot IDs and does not reverse-map for outbound —
// callers still publish via AGV(mfr, sn).
type IdentityMapper func(mfr, serial string) string

// String returns manufacturer/serial for logging.
func (id Identity) String() string {
	return id.Manufacturer + "/" + id.SerialNumber
}
