package auth

import (
	"back-rex-common/pkg/services"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	b        int
}

func newRateLimiter(r rate.Limit, b int) *rateLimiter {
	rl := &rateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		b:        b,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.limiters[ip]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(rl.r, rl.b)}
		rl.limiters[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// supprime les IPs inactives depuis plus de 10 minutes
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, entry := range rl.limiters {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// 5 tentatives par minute par IP
var loginRateLimiter = newRateLimiter(rate.Every(time.Minute/5), 5)

func RateLimitLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			services.AuthenticationError(w, r, "pas de header X-Real-IP", services.NO_INFORMATION, nil)
			return
		}

		if !loginRateLimiter.get(ip).Allow() {
			services.AuthenticationError(w, r, "trop de tentatives, réessayez plus tard", services.NO_INFORMATION, nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
