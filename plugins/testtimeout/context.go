package testtimeout

import (
	"context"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

func applyContextDeadline(c *axiom.Config, deadline time.Time) func() {
	c.Context.Normalize()

	var cancelRaw, cancelDB, cancelMQ, cancelRPC context.CancelFunc
	c.Context.Raw, cancelRaw = context.WithDeadline(c.Context.Raw, deadline)
	c.Context.DB, cancelDB = context.WithDeadline(c.Context.DB, deadline)
	c.Context.MQ, cancelMQ = context.WithDeadline(c.Context.MQ, deadline)
	c.Context.RPC, cancelRPC = context.WithDeadline(c.Context.RPC, deadline)

	return func() {
		cancelRaw()
		cancelDB()
		cancelMQ()
		cancelRPC()
	}
}
