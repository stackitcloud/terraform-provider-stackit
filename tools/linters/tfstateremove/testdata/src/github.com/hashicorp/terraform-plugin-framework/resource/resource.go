package resource

import "context"

type ReadRequest struct{}

type ReadResponse struct {
	State *State
}

type State struct{}

func (*State) RemoveResource(context.Context) {}
