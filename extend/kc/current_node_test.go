package kc_test

import (
	"testing"

	"github.com/kalifun/navlink/extend"
	"github.com/kalifun/navlink/extend/kc"
)

func TestCurrentNodeID(t *testing.T) {
	reg := extend.NewRegistry()
	reg.Register(kc.CurrentNodeID())

	meta, err := reg.Apply("state", []byte(`{"currentNodeId":"N42","lastNodeId":"N1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if meta[kc.MetaReportedCurrentNodeID] != "N42" {
		t.Fatalf("meta=%v", meta)
	}

	meta, err = reg.Apply("connection", []byte(`{"currentNodeId":"N42"}`))
	if err != nil || len(meta) != 0 {
		t.Fatalf("connection should ignore: meta=%v err=%v", meta, err)
	}
}
