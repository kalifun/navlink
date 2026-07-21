package topic_test

import (
	"testing"

	"github.com/kalifun/navlink/topic"
)

func TestResolverBuildParse(t *testing.T) {
	r := topic.Resolver{Interface: "uagv", Version: "v2"}

	got := r.Build("RobotCorp", "AGV001", topic.ChannelState)
	want := "uagv/v2/RobotCorp/AGV001/state"
	if got != want {
		t.Fatalf("Build = %q, want %q", got, want)
	}

	parsed, err := r.Parse(got)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.Interface != "uagv" || parsed.Version != "v2" {
		t.Fatalf("interface/version = %s/%s", parsed.Interface, parsed.Version)
	}
	if parsed.Manufacturer != "RobotCorp" || parsed.SerialNumber != "AGV001" {
		t.Fatalf("identity = %s/%s", parsed.Manufacturer, parsed.SerialNumber)
	}
	if parsed.Channel != topic.ChannelState {
		t.Fatalf("channel = %s", parsed.Channel)
	}
}

func TestResolverWildcard(t *testing.T) {
	r := topic.Resolver{Interface: "vda5050", Version: "v2.0.0"}
	got := r.Wildcard(topic.ChannelConnection)
	want := "vda5050/v2.0.0/+/+/connection"
	if got != want {
		t.Fatalf("Wildcard = %q, want %q", got, want)
	}
}

func TestResolverParseErrors(t *testing.T) {
	r := topic.Resolver{Interface: "uagv", Version: "v2"}
	cases := []string{
		"too/short",
		"uagv/v2/mfr/sn/unknown",
		"uagv/v2/mfr/sn/state/extra",
	}
	for _, tc := range cases {
		if _, err := r.Parse(tc); err == nil {
			t.Fatalf("Parse(%q) want error", tc)
		}
	}
}
