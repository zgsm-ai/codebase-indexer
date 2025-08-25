package daemon

import "context"

type Job interface {
	Start(ctx context.Context)
}
