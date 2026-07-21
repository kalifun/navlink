package navlink

import (
	"context"

	"github.com/kalifun/vda5050-types-go/connection"
	"github.com/kalifun/vda5050-types-go/factsheet"
	"github.com/kalifun/vda5050-types-go/state"
	"github.com/kalifun/vda5050-types-go/visualization"
)

// StateHandler handles a decoded state message.
type StateHandler func(ctx context.Context, env Envelope, msg *state.State) error

// ConnectionHandler handles a decoded connection message.
type ConnectionHandler func(ctx context.Context, env Envelope, msg *connection.Connection) error

// VisualizationHandler handles a decoded visualization message.
type VisualizationHandler func(ctx context.Context, env Envelope, msg *visualization.Visualization) error

// FactsheetHandler handles a decoded factsheet message.
type FactsheetHandler func(ctx context.Context, env Envelope, msg *factsheet.Factsheet) error

// TopicHandler is the escape hatch for non-typed topic filters.
type TopicHandler func(ctx context.Context, env Envelope) error

// DecodeErrorHandler observes decode/identity failures without crashing the process.
type DecodeErrorHandler func(env Envelope, err error)
