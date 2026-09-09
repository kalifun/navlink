package mqtt

import "testing"

func TestClassifyInbound(t *testing.T) {
	tests := []struct {
		topic string
		lane  inboundLane
		key   string
	}{
		{"uagv/v2/M/S1/connection", laneConnection, ""},
		{"uagv/v2/M/S1/state", laneAGV, "M/S1"},
		{"uagv/v2/M/S1/visualization", laneAGV, "M/S1"},
		{"uagv/v2/M/S1/factsheet", laneAGV, "M/S1"},
		{"uagv/v2/M/S2/state", laneAGV, "M/S2"},
		{"app/custom/topic", laneTopic, ""},
		{"short", laneTopic, ""},
	}
	for _, tc := range tests {
		lane, key := classifyInbound(tc.topic)
		if lane != tc.lane || key != tc.key {
			t.Fatalf("%s: lane=%v key=%q want lane=%v key=%q", tc.topic, lane, key, tc.lane, tc.key)
		}
	}
}
