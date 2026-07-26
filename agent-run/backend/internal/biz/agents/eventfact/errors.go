package eventfact

import "fmt"

type RemoteError struct {
	Code      string
	Summary   string
	Retryable bool
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Summary)
}
