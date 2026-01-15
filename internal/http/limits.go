package http

import (
	"time"
	"golang.org/x/time/rate"
)
var (
	LoginLimiter    = NewIPRateLimiter(rate.Every(time.Minute/5), 5)
	RegisterLimiter = NewIPRateLimiter(rate.Every(time.Minute/3), 3)
	EmailLimiter    = NewIPRateLimiter(rate.Every(time.Minute/3), 3)
	GlobalLimiter   = NewIPRateLimiter(rate.Every(time.Second), 60)
)

