package deepseek

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestExecutionBoundClientCancelsProviderRequestWithExecution(t *testing.T) {
	executionContext, cancelExecution := context.WithCancel(context.Background())
	requestStarted := make(chan struct{})
	client := executionBoundClient(executionContext, 10*time.Second, roundTripperFunc(
		func(request *http.Request) (*http.Response, error) {
			close(requestStarted)
			<-request.Context().Done()
			return nil, request.Context().Err()
		},
	))
	result := make(chan error, 1)
	go func() {
		request, _ := http.NewRequest(http.MethodPost, "https://deepseek.test/chat/completions", nil)
		_, err := client.Do(request)
		result <- err
	}()
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("Provider request did not start")
	}
	cancelExecution()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Provider request error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Provider request was not canceled with the Execution")
	}
}
