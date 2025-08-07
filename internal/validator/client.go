package validator

import (
	"context"
)

// Client interface for validator clients
type Client interface {
	GetNodeInfo(ctx context.Context) (*ValidatorNodeInfo, error)
}
