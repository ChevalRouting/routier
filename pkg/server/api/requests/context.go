package requests

import (
	"context"
	"net/http"
)

func DurableContext(r *http.Request) context.Context {
	return context.WithoutCancel(r.Context())
}
