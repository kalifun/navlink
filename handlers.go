package navlink

import (
	"context"

	"github.com/kalifun/vda5050-types-go/connection"
	"github.com/kalifun/vda5050-types-go/factsheet"
	"github.com/kalifun/vda5050-types-go/state"
	"github.com/kalifun/vda5050-types-go/visualization"
)

// StateHandler handles a decoded state message.
// It runs on the inbound worker and must return quickly: no HTTP, no Subscribe,
// no long locks, no waiting for MQTT Publish. Do slow work in another goroutine.
type StateHandler func(ctx context.Context, env Envelope, msg *state.State) error

// ConnectionHandler handles a decoded connection message.
// Same rules as StateHandler. Do not Client.Track / Subscribe here; use a
// platform queue or explicit Track outside the inbound path.
type ConnectionHandler func(ctx context.Context, env Envelope, msg *connection.Connection) error

// VisualizationHandler handles a decoded visualization message.
type VisualizationHandler func(ctx context.Context, env Envelope, msg *visualization.Visualization) error

// FactsheetHandler handles a decoded factsheet message.
type FactsheetHandler func(ctx context.Context, env Envelope, msg *factsheet.Factsheet) error

// TopicHandler is the escape hatch for non-typed topic filters.
// Same threading rules as StateHandler. On the built-in MQTT transport, custom
// topics share an isolated lane (not the per-AGV state shards).
type TopicHandler func(ctx context.Context, env Envelope) error

// DecodeErrorHandler observes decode/identity failures without crashing the process.
type DecodeErrorHandler func(env Envelope, err error)

// HandlerErrorHandler observes inbound handler errors without crashing the process.
type HandlerErrorHandler func(env Envelope, err error)
