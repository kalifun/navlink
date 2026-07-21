package navlink

// Identity is the protocol-level AGV identity (manufacturer + serial).
type Identity struct {
	Manufacturer string
	SerialNumber string
}

// IdentityMapper optionally maps protocol identity to a platform robot ID.
type IdentityMapper func(mfr, serial string) string

// String returns manufacturer/serial for logging.
func (id Identity) String() string {
	return id.Manufacturer + "/" + id.SerialNumber
}
