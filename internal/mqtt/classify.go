package mqtt

import "strings"

type inboundLane int

const (
	laneConnection inboundLane = iota
	laneAGV
	laneTopic
)

// classifyInbound routes VDA topics onto connection / per-AGV / other lanes.
// Topic shape: interface/version/manufacturer/serial/channel.
func classifyInbound(topic string) (lane inboundLane, shardKey string) {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		return laneTopic, ""
	}
	channel := parts[len(parts)-1]
	mfr := parts[len(parts)-3]
	sn := parts[len(parts)-2]
	switch channel {
	case "connection":
		return laneConnection, ""
	case "state", "visualization", "factsheet":
		return laneAGV, mfr + "/" + sn
	default:
		return laneTopic, ""
	}
}

func droppableTopic(topic string) bool {
	return strings.HasSuffix(topic, "/visualization")
}
