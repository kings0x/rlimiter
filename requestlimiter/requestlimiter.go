package requestlimiter

import (
	"context"
	"time"

	"github.com/kings0x/rlimiter/engine"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(r *redis.Client) *RedisStore {
	return &RedisStore{
		client: r,
	}
}

const luascript = `
	local key = KEYS[1]
	local rate = tonumber(ARGV[1])
	local capacity = tonumber(ARGV[2])
	local tokenCost = tonumber(ARGV[3])
	local now = tonumber(ARGV[4])

	local data = redis.call("HMGET", key, "tokens", "last")
	local tokens = tonumber(data[1])
	local last = tonumber(data[2])

	if tokens == nil then
		tokens = capacity
		last = now
	end

	local elapsed = now - last
	if elapsed > 0 then 
		tokens = math.min(capacity, tokens + elapsed * rate)
	end

	local allowed = 0

	if tokens >= tokenCost then
		tokens = tokens - tokenCost
		allowed = 1
	end

	redis.call("HMSET", key, "tokens", tokens, "last", now)
	redis.call("EXPIRE", key, math.ceil(capacity/rate))

	return {allowed, tokens}

`

type RequestLimiter struct {
	rate      float64
	capacity  float64
	tokenCost float64
	name      string
	store     *RedisStore
}

type Options struct {
	Rate      float64
	Capacity  float64
	TokenCost float64
	Name      string
	Store     *RedisStore
}

func New(opts Options) *RequestLimiter {
	if opts.Store == nil {
		panic("RedisStore required")
	}

	if opts.TokenCost <= 0 {
		opts.TokenCost = 1
	}

	name := opts.Name

	if name == "" {
		name = "request"
	}

	return &RequestLimiter{
		rate:      opts.Rate,
		capacity:  opts.Capacity,
		tokenCost: opts.TokenCost,
		name:      name,
		store:     opts.Store,
	}
}

func (r *RequestLimiter) Name() string {
	return r.name
}

func (r *RequestLimiter) Allow(key string) engine.Result {
	if key == "" {
		key = "global"
	}

	ctx := context.Background()

	now := float64(time.Now().UnixNano()) / 1e9

	res, err := r.store.client.Eval(ctx, luascript, []string{key}, r.rate, r.capacity, r.tokenCost, now).Result()

	if err != nil {
		//if redis fails we do not crash the ops, we just keep on going.
		return engine.Result{
			Name:      r.name,
			Allowed:   true,
			Remaining: r.capacity,
		}
	}

	arr := res.([]interface{})
	allowed := arr[0].(int64) == 1
	remaining := arr[1].(float64)

	var retryAfter time.Duration

	if !allowed {
		need := r.tokenCost - remaining

		if need < 0 {
			need = 0
		}

		if r.rate > 0 {
			secs := need / r.rate
			retryAfter = time.Duration(secs * float64(time.Second))
		}
	}

	return engine.Result{
		Name:       r.name,
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: retryAfter,
	}
}
