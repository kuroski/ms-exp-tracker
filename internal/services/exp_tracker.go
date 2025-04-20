package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kuroski/ms-exp-tracker/internal/clock"
	"github.com/kuroski/ms-exp-tracker/internal/utils"
)

const (
	DefaultTickerInterval = 5 * time.Second
	MaxMeasurements       = 100
)

type Measurement struct {
	exp        int
	percentage float64
	createdAt  time.Time
}

type Stats struct {
	// xpPerMin := xpPerSecond * 60
	// xpPerHour := xpPerSecond * 3600
	// fmt.Printf("XP Rate: %.2f XP/min, %.2f XP/hour\n", xpPerMin, xpPerHour)
	ExpPerSecond float64
	// percPerMin := percPerSecond * 60
	// percPerHour := percPerSecond * 3600
	// fmt.Printf("Percentage Rate: %.2f%%/min, %.2f%%/hour\n", percPerMin, percPerHour)
	PercentPerSecond float64
	// fmt.Printf("Time to Level Up: %.2f minutes (%.2f hours)\n", timeToLevelUp/60, timeToLevelUp/3600)
	TimeToLevelUpPerSecond float64
	CurrentXp              int
	LevelProgress          float64
}

type TrackResult struct {
	Stats Stats
	Err   error
}

type ExpTracker struct {
	Ch chan TrackResult
	mu sync.Mutex

	measurements []Measurement

	ctx    context.Context
	cancel context.CancelFunc

	args Args
}

type Args struct {
	ExpCrawler ExpCrawler
	Interval   time.Duration
	Clock      clock.Clock
	Logger     *slog.Logger
}

func NewExpTracker(args Args) *ExpTracker {
	if args.Interval <= 0 {
		args.Interval = DefaultTickerInterval
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ExpTracker{
		ctx:    ctx,
		cancel: cancel,
		Ch:     make(chan TrackResult, 1),
		args:   args,
	}
}

func (t *ExpTracker) Run() {
	ticker := time.NewTicker(t.args.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			t.args.Logger.Info("Ticker has been finished")
			return
		case <-ticker.C:
			if err := t.recordMeasurement(); err != nil {
				t.sendError(err)
			}

			t.sendStats()
		}
	}
}

func (t *ExpTracker) Stop() {
	t.cancel()
	close(t.Ch)
}

func (t *ExpTracker) sendError(err error) {
	select {
	case t.Ch <- TrackResult{Err: err}:
		t.args.Logger.Error("An error ocurred while recoding a measurement", "error:", err)
	default:
		t.args.Logger.Debug("Skipped sending error due to full channel")
	}
}

func (t *ExpTracker) sendStats() {
	trackResult := TrackResult{Stats: t.getStats()}
	select {
	case t.Ch <- trackResult:
		t.args.Logger.Info("Tick measurement", "stats:", trackResult.Stats)
	default:
		t.args.Logger.Debug("Skipped sending error due to full channel")
	}
}

func (t *ExpTracker) recordMeasurement() error {
	crawlResult, err := t.args.ExpCrawler.Crawl()
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Prevent duplicate measurements
	if last, ok := utils.Last(t.measurements); ok {
		if last.exp == crawlResult.Exp || last.percentage == crawlResult.Percentage {
			return nil
		}
	}

	t.measurements = append(t.measurements, Measurement{
		exp:        crawlResult.Exp,
		percentage: crawlResult.Percentage,
		createdAt:  time.Now(),
	})

	if len(t.measurements) > MaxMeasurements {
		t.measurements = t.measurements[len(t.measurements)-MaxMeasurements:]
	}

	return nil
}

func (t *ExpTracker) getStats() Stats {
	t.mu.Lock()
	defer t.mu.Unlock()

	latest, ok := utils.Last(t.measurements)
	if !ok {
		return Stats{}
	}

	xpPerSecond, percentagePerSecond := t.calculateXpRate()

	if xpPerSecond <= 0 {
		return Stats{}
	}

	return Stats{
		ExpPerSecond:           xpPerSecond,
		PercentPerSecond:       percentagePerSecond,
		TimeToLevelUpPerSecond: t.estimateTimeToLevelUp(latest.percentage, percentagePerSecond),
		CurrentXp:              latest.exp,
		LevelProgress:          latest.percentage,
	}
}

func (t *ExpTracker) calculateXpRate() (xpPerSecond, percentagePerSecond float64) {
	if len(t.measurements) < 2 {
		return 0, 0
	}

	first := t.measurements[len(t.measurements)-2]
	last, _ := utils.Last(t.measurements)

	expGained := float64(last.exp - first.exp)
	percentageGained := last.percentage - first.percentage

	// timeElapsed := last.createdAt.Sub(first.createdAt).Seconds()
	timeElapsed := t.args.Clock.Since(first.createdAt).Seconds()
	if timeElapsed <= 0 {
		return 0, 0
	}

	return expGained / timeElapsed, percentageGained / timeElapsed
}

func (t *ExpTracker) estimateTimeToLevelUp(currentPercentage, percentagePerSecond float64) float64 {
	remaining := 100.0 - currentPercentage
	if percentagePerSecond <= 0 {
		return -1
	}

	return remaining / percentagePerSecond
}
