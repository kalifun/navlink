// Package navlink is a VDA5050 protocol access SDK for Go.
//
// It provides MQTT connectivity, a single TopicResolver, typed inbound
// handlers (OnState / OnConnection / …), and typed outbound publishing via
// AGVHandle. Scheduling and domain orchestration stay outside this package.
//
// See docs/PRODUCT_SPEC.md for product boundaries and milestones.
package navlink
