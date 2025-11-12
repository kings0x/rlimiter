package engine

import "time"

//later on understand what each of these do

type Result struct {
	Name string

	Allowed bool

	Remaining float64

	RetryAfter time.Duration
}

//this is a limiter interface all the types of limiters implement this
//basically the methods each limiter would implement
type Limiter interface {
	Name() string

	Allow(key string) Result
}

//engine
// Engine composes multiple limiters. It runs them in order and
// returns the first denying result (fail-fast). If all allow, returns an allowed Result.

type Engine struct {
	limiters []Limiter
}

// New creates an engine composed of the given limiters (order matters).

func New(limiters ...Limiter) *Engine {
	return &Engine{limiters: limiters}
}

// Allow runs each limiter; stops at first denial.

func (e *Engine) Allow(key string) Result {
	var aggregateRemaining float64

	for _, l := range e.limiters {
		res := l.Allow(key)

		if !res.Allowed {
			return res
		}

		aggregateRemaining += res.Remaining
	}

	return Result{
		Name:      "composite",
		Allowed:   true,
		Remaining: aggregateRemaining,
	}
}
