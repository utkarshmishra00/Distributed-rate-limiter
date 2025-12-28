package limiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client   *redis.Client
	rate     int // tokens per second
	capacity int // max burst or capacity
}

func NewRateLimiter(client *redis.Client, rate int, capacity int) *RateLimiter {
	return &RateLimiter{
		client:   client,
		rate:     rate,
		capacity: capacity,
	}
}

func (rl *RateLimiter) Allow(key string) (bool, int64, int64, int64, error) {
	ctx := context.Background()

	const script = `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = 1

		local last_checked = tonumber(redis.call('hget', key, 'last_checked') or 0)
		local tokens = tonumber(redis.call('hget', key, 'tokens') or capacity)

		local elapsed_millis = math.max(0, now - last_checked)
		local elapsed_seconds = elapsed_millis / 1000
		local new_tokens = math.min(capacity, tokens + (elapsed_seconds * rate))

		local allowed = 0
		local retry_after_ms = 0 -- Default to 0

		if new_tokens >= requested then
			new_tokens = new_tokens - requested
			allowed = 1
		else
			local deficit = requested - new_tokens
			-- Calculate wait time in Seconds, then convert to MS
			local wait_seconds = deficit / rate
			retry_after_ms = math.ceil(wait_seconds * 1000)
		end

		local tokens_needed = capacity - new_tokens
		local time_to_full_seconds = tokens_needed / rate
		local reset_time_millis = now + (time_to_full_seconds * 1000)

		redis.call('hset', key, 'tokens', new_tokens)
		redis.call('hset', key, 'last_checked', now)
		redis.call('expire', key, 60)

		-- Return everything as INTEGERS (Safe for Redis Protocol)
		return { allowed, new_tokens, reset_time_millis / 1000, retry_after_ms }
	`

	now := time.Now().UnixMilli()

	cmd := rl.client.Eval(ctx, script, []string{key}, rl.rate, rl.capacity, now)
	result, err := cmd.Result()
	if err != nil {
		return false, 0, 0, 0, err
	}

	resSlice := result.([]interface{})

	//safely cast any number to int64
	toInt64 := func(val interface{}) int64 {
		switch v := val.(type) {
		case int64:
			return v
		case float64:
			return int64(v)
		default:
			return 0
		}
	}

	allowedVal := toInt64(resSlice[0])
	tokensLeft := toInt64(resSlice[1])
	resetTime := toInt64(resSlice[2])
	retryAfterMs := toInt64(resSlice[3])

	return allowedVal == 1, tokensLeft, resetTime, retryAfterMs, nil
}
