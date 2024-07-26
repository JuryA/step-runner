package resource

import "context"

type Resource interface {
	Load(context.Context) (string, error)
}
