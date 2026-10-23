package utils

import (
	"net/http"
	"time"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

var RetryConfig = utils.RetryConfig{
	Attempts: 5,
	Backoff: func(attempt int) time.Duration {
		// Wait for every attempt 5 seconds longer. 5s, 10s, 15s and so on
		return time.Duration(attempt*5) * time.Second
	},
	RetryStatusCodes: []int{
		http.StatusLocked,
		http.StatusTooEarly,
	},
}
