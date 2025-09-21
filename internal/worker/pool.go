package worker

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"gopool/internal/backoff"
	"gopool/internal/queue"
	"gopool/internal/simulation"
)

type Sleeper interface{ Sleep(time.Duration) }
type Runner func(t queue.Task) error
type Backoff func(attempt int, r *rand.Rand) time.Duration

// Config описывает настройки пула воркеров.
type Config struct {
	Workers int
	Sleeper Sleeper
	Backoff Backoff
	RNG     *rand.Rand
	Runner  Runner
}

type Option func(*Config)

func WithWorkers(n int) Option     { return func(c *Config) { c.Workers = n } }
func WithSleeper(s Sleeper) Option { return func(c *Config) { c.Sleeper = s } }
func WithRNG(r *rand.Rand) Option  { return func(c *Config) { c.RNG = r } }
func WithBackoff(b Backoff) Option { return func(c *Config) { c.Backoff = b } }

// WithBackoffParams настраивает бэкофф через backoff.Delay с base/maxDelay
func WithBackoffParams(base, maxDelay time.Duration) Option {
	return func(c *Config) {
		c.Backoff = func(attempt int, r *rand.Rand) time.Duration {
			return backoff.Delay(attempt, base, maxDelay, r)
		}
	}
}

func WithRunner(r Runner) Option { return func(c *Config) { c.Runner = r } }

func NewPool(opts ...Option) *Pool {
	cfg := Config{
		Workers: 4,
		Sleeper: RealSleeper{},
		Backoff: func(attempt int, r *rand.Rand) time.Duration {
			return backoff.Delay(attempt, 50*time.Millisecond, 2*time.Second, r)
		},
		RNG:    nil,
		Runner: nil,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Pool{cfg: cfg}
}

type Pool struct {
	cfg Config
}

// Run запускает воркеры и блокируется до отмены контекста или дренажа очереди
func (p *Pool) Run(ctx context.Context, q *queue.Q, store queue.Store) {
	n := p.effectiveWorkers()
	seeds := deriveSeeds(p.cfg.RNG, n)

	var wg sync.WaitGroup
	wg.Add(n)

	workersDone := make(chan struct{})

	for i := 0; i < n; i++ {
		rng := rand.New(rand.NewSource(seeds[i]))
		go func(r *rand.Rand) {
			defer wg.Done()
			p.workerLoop(ctx, q, store, r)
		}(rng)
	}

	go func() {
		wg.Wait()
		close(workersDone)
	}()

	select {
	case <-ctx.Done():
		<-workersDone
	case <-workersDone:
	}
}

// workerLoop обрабатывает задачи из очереди до завершения.
func (p *Pool) workerLoop(ctx context.Context, q *queue.Q, store queue.Store, r *rand.Rand) {
	runner := p.cfg.Runner
	if runner == nil {
		runner = DefaultRunnerFor(r, p.cfg.Sleeper)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case t, ok := <-q.Take():
			if !ok {
				return
			}
			processTask(ctx, t, store, runner, p.cfg.Backoff, p.cfg.Sleeper, r)
		}
	}
}

func DefaultRunnerFor(r *rand.Rand, sl Sleeper) Runner {
	return func(t queue.Task) error {
		sl.Sleep(simulation.SimulatedDuration(r))
		if simulation.ShouldFail(r) {
			return ErrSimulated
		}
		return nil
	}
}

// processTask выполняет задачу с ретраями и бэкоффом до MaxRetries.
func processTask(
	ctx context.Context,
	t queue.Task,
	store queue.Store,
	runner Runner,
	backoff Backoff,
	sl Sleeper,
	r *rand.Rand,
) {
	store.SetState(t.ID, queue.StateRunning)

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := runner(t); err == nil {
			store.SetState(t.ID, queue.StateDone)
			return
		}

		attempt++
		if attempt > t.MaxRetries {
			store.SetState(t.ID, queue.StateFailed)
			return
		}

		delay := backoff(attempt, r)
		if delay <= 0 {
			continue
		}
		if !sleepWithContext(ctx, sl, delay) {
			return
		}
	}
}

type RealSleeper struct{}

func (RealSleeper) Sleep(d time.Duration) { time.Sleep(d) }

var ErrSimulated = errors.New("simulated failure")

// effectiveWorkers возвращает валидное число воркеров (минимум 1).
func (p *Pool) effectiveWorkers() int {
	if p.cfg.Workers <= 0 {
		return 1
	}
	return p.cfg.Workers
}

// deriveSeeds генерирует сиды для per-worker RNG.
func deriveSeeds(base *rand.Rand, n int) []int64 {
	seeds := make([]int64, n)
	if base != nil {
		for i := 0; i < n; i++ {
			seeds[i] = base.Int63()
		}
		return seeds
	}
	now := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		seeds[i] = now + int64(i)*1_000_003
	}
	return seeds
}

// sleepWithContext делает задержку с возможностью отмены через ctx.
func sleepWithContext(ctx context.Context, sl Sleeper, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	done := make(chan struct{}, 1)
	go func() {
		sl.Sleep(d)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		<-done
		return true
	}
}
