package apitest

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/ChevalRouting/routier/pkg/server/api/requests"
)

type contextKey string

func TestDurableContextKeepsValuesAndDropsCancellation(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey("request"), "value"))
	request := httptest.NewRequest("POST", "/", nil).WithContext(requestContext)
	durable := requests.DurableContext(request)

	cancel()

	if durable.Err() != nil {
		t.Fatalf("durable context canceled: %v", durable.Err())
	}

	if got := durable.Value(contextKey("request")); got != "value" {
		t.Fatalf("request value = %v, want value", got)
	}

	if _, ok := durable.Deadline(); ok {
		t.Fatal("durable context retained request deadline")
	}
}
