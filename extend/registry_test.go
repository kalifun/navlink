package extend_test

import (
	"encoding/json"
	"testing"

	"github.com/kalifun/navlink/extend"
)

func TestRegistryMergesConsumerExtractors(t *testing.T) {
	reg := extend.NewRegistry()
	reg.Register(func(channel string, raw []byte) (extend.Meta, error) {
		if channel != "state" {
			return nil, nil
		}
		var probe struct {
			Extra string `json:"extraField"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if probe.Extra == "" {
			return nil, nil
		}
		return extend.Meta{"ExtraField": probe.Extra}, nil
	})

	meta, err := reg.Apply("state", []byte(`{"extraField":"ok","lastNodeId":"N1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if meta["ExtraField"] != "ok" {
		t.Fatalf("meta=%v", meta)
	}
}
