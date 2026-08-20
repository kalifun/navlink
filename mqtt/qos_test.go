package mqtt

import "testing"

func TestSubscribeQoSDefaults(t *testing.T) {
	tr := &Transport{cfg: Config{QoS: 1, QoSExplicit: false}}
	if q := tr.subscribeQoS("uagv/v2/+/+/state"); q != 1 {
		t.Fatalf("state qos=%d", q)
	}
	if q := tr.subscribeQoS("uagv/v2/+/+/visualization"); q != 0 {
		t.Fatalf("viz qos=%d", q)
	}
	tr.cfg.QoSExplicit = true
	tr.cfg.QoS = 0
	if q := tr.subscribeQoS("uagv/v2/+/+/visualization"); q != 0 {
		t.Fatalf("explicit 0 viz=%d", q)
	}
	if q := tr.subscribeQoS("uagv/v2/+/+/state"); q != 0 {
		t.Fatalf("explicit 0 state=%d", q)
	}
}
