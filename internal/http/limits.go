package http

import (
	"time"

	"golang.org/x/time/rate"
)

var (
	LoginLimiter    = NewKeyedRateLimiter(rate.Every(time.Minute/5), 5)
	RegisterLimiter = NewKeyedRateLimiter(rate.Every(time.Minute/3), 3)
	EmailLimiter    = NewKeyedRateLimiter(rate.Every(time.Minute/3), 3)
	GlobalLimiter   = NewKeyedRateLimiter(rate.Every(time.Second), 60)
)
