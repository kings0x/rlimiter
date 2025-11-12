package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kings0x/rlimiter/engine"
)

type KeyFunc func(r *http.Request) string

func DefaultKeyByIp(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		parts := strings.Split(f, ",")

		return strings.TrimSpace(parts[0])
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	if ip == "" {
		return "global"
	}
	return ip
}

type contextKey string

const limiterResultContextKey = contextKey("ratelimit.result")

func ResultFromContext(ctx context.Context) (engine.Result, bool) {
	res, ok := ctx.Value(limiterResultContextKey).(engine.Result)

	return res, ok
}

func New(e *engine.Engine, keyFn KeyFunc) func(http.Handler) http.Handler {
	if keyFn == nil {
		keyFn = DefaultKeyByIp
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			res := e.Allow(key)

			w.Header().Set("X-RateLimit-Limit", strconv.FormatFloat(res.Remaining+0, 'f', -1, 64))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatFloat(res.Remaining, 'f', -1, 64))

			if res.RetryAfter > 0 {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(res.RetryAfter/time.Second), 10))
			}

			ctx := context.WithValue(r.Context(), limiterResultContextKey, res)

			r = r.WithContext(ctx)

			if !res.Allowed {
				http.Error(w, "rate limit  exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
