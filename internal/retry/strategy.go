package retry

import (
	"time"

	"github.com/wb-go/wbf/retry"
)

var DefaultStrategy = retry.Strategy{
	Attempts: 3,
	Delay:    100 * time.Millisecond,
	Backoff:  2.0,
}
