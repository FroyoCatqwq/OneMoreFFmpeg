package model

// capabilityKindOrder is the authoritative list of --only values. It lives in
// model (a zero-dependency DTO package) so contracts can validate capability
// examples without importing the capabilities implementation package.
var capabilityKindOrder = [...]string{
	"muxers", "demuxers", "devices", "decoders", "encoders", "filters",
	"pixel_formats", "channel_layouts", "sample_formats",
}

// capabilityFullCommandKeys is the authoritative set of FFmpeg command keys
// reported in a full capabilities result. Keep in sync with the commandSpecs
// table in internal/capabilities.
var capabilityFullCommandKeys = [...]string{
	"-L", "-license", "-h ", "-version", "-muxers", "-demuxers",
	"-devices", "-decoders", "-encoders", "-filters", "-pix_fmts",
	"-layouts", "-sample_fmts",
}

// IsCapabilityKind reports whether kind is a supported single-list query.
func IsCapabilityKind(kind string) bool {
	for _, k := range capabilityKindOrder {
		if k == kind {
			return true
		}
	}
	return false
}

// CapabilityKinds returns the supported --only values in a stable order.
// A copy is returned so callers cannot mutate the authoritative list.
func CapabilityKinds() []string {
	out := make([]string, len(capabilityKindOrder))
	copy(out, capabilityKindOrder[:])
	return out
}

// CapabilityFullCommandKeys returns the FFmpeg command keys in a full
// capabilities result. A copy is returned so callers cannot mutate it.
func CapabilityFullCommandKeys() []string {
	out := make([]string, len(capabilityFullCommandKeys))
	copy(out, capabilityFullCommandKeys[:])
	return out
}
