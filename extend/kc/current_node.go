package kc

import (
	"encoding/json"

	"github.com/kalifun/navlink/extend"
)

// MetaReportedCurrentNodeID is the Envelope.Meta key for KC currentNodeId.
const MetaReportedCurrentNodeID = "ReportedCurrentNodeID"

// CurrentNodeID extracts the KC vendor field currentNodeId from state payloads.
// OEM error codes and disposition policy stay outside navlink.
func CurrentNodeID() extend.Extractor {
	return func(channel string, raw []byte) (extend.Meta, error) {
		if channel != "state" {
			return nil, nil
		}
		var probe struct {
			CurrentNodeID string `json:"currentNodeId"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if probe.CurrentNodeID == "" {
			return nil, nil
		}
		return extend.Meta{MetaReportedCurrentNodeID: probe.CurrentNodeID}, nil
	}
}
