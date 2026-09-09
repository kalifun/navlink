package navlink

import (
	"context"
	"errors"

	"github.com/kalifun/vda5050-types-go/connection"

	"github.com/kalifun/navlink/internal/session"
)

const fleetAutoTrackQueueSize = 64

var errFleetAutoTrackQueueFull = errors.New("navlink: fleet auto-track queue full")

type fleetConnJob struct {
	agv   session.AGV
	state connection.ConnectionState
	env   Envelope
}

func (c *Client) startFleetAutoTrack() {
	if c.fleet == nil || !c.fleet.AutoTrackFromConnection() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.fleetStop = cancel
	c.fleetDone = ctx.Done()
	c.fleetJobs = make(chan fleetConnJob, fleetAutoTrackQueueSize)
	c.fleetWG.Go(func() {
		c.runFleetAutoTrack(ctx)
	})
}

func (c *Client) runFleetAutoTrack(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-c.fleetJobs:
			if !ok {
				return
			}
			if err := c.fleet.HandleConnection(ctx, job.agv, job.state); err != nil {
				c.reportFleetTrackError(job.env, err)
			}
		}
	}
}

func (c *Client) stopFleetAutoTrack() {
	if c.fleetStop != nil {
		c.fleetStop()
		c.fleetStop = nil
	}
	c.fleetWG.Wait()
	c.mu.Lock()
	c.fleetJobs = nil
	c.fleetDone = nil
	c.mu.Unlock()
}

// scheduleAutoTrack enqueues opt-in connection→Track work without blocking the
// inbound typed path. No-op when Fleet is off or AutoTrackFromConnection is false.
func (c *Client) scheduleAutoTrack(env Envelope, state connection.ConnectionState) {
	if c.fleet == nil || !c.fleet.AutoTrackFromConnection() {
		return
	}
	c.mu.RLock()
	jobs := c.fleetJobs
	done := c.fleetDone
	c.mu.RUnlock()
	if jobs == nil {
		return
	}
	job := fleetConnJob{
		agv:   sessionAGV(env.AGV),
		state: state,
		env:   env,
	}
	select {
	case jobs <- job:
	case <-done:
	default:
		c.reportFleetTrackError(env, errFleetAutoTrackQueueFull)
	}
}

func (c *Client) reportFleetTrackError(env Envelope, err error) {
	c.mu.RLock()
	h := c.onHandlerError
	c.mu.RUnlock()
	if h != nil {
		h(env, err)
	}
}
