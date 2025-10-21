package runner

import (
	"context"
)

type RunnerLocal struct {
	ctx     context.Context
	options *Options
}

func NewRunnerLocal(
	ctx context.Context,
	options *Options,
) (*RunnerLocal, error) {
	return &RunnerLocal{
		ctx:     ctx,
		options: options,
	}, nil
}

func (c *RunnerLocal) Initialize() error {
	return nil
}
