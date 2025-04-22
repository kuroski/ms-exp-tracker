package services

import (
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/kuroski/ms-exp-tracker/internal/clock"
	"github.com/kuroski/ms-exp-tracker/internal/logger"
)

type MockExpCrawler struct {
	exp        int
	percentage float64
	err        error
}

func (m *MockExpCrawler) Crawl() (result CrawlResult, err error) {
	return CrawlResult{
		Exp:        m.exp,
		Percentage: m.percentage,
	}, m.err
}

type fakeClock struct {
	currentTime time.Time
}

func newFakeClock() clock.Clock {
	return &fakeClock{
		currentTime: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
	}
}

func (c *fakeClock) Now() time.Time {
	return c.currentTime
}

func (c *fakeClock) Since(t time.Time) time.Duration {
	return c.currentTime.Sub(t)
}

func (c *fakeClock) Advance(d time.Duration) {
	c.currentTime = c.currentTime.Add(d)
}

func TestExpTracker_RunAndStop(t *testing.T) {
	interval := 100 * time.Millisecond
	expCrawler := &MockExpCrawler{exp: 1000, percentage: 50}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	args := Args{
		ExpCrawler: expCrawler,
		Interval:   interval,
		Clock:      newFakeClock(),
		Logger:     logger,
	}
	tracker := NewExpTracker(args)

	done := make(chan struct{})

	go func() {
		tracker.Run()
		close(done)
	}()

	// Allow some time for the tracker to perform measurements.
	time.Sleep(250 * time.Millisecond)

	tracker.Stop()

	// Wait for the goroutine to exit
	select {
	case <-done:
		// Goroutine exited successfully
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timed out waiting for goroutine to exit")
	}
}

func TestExpTracker_ErrorHandling(t *testing.T) {
	interval := 100 * time.Millisecond
	expCrawler := &MockExpCrawler{
		err: fmt.Errorf("crawler error"),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	args := Args{
		ExpCrawler: expCrawler,
		Interval:   interval,
		Clock:      newFakeClock(),
		Logger:     logger,
	}
	tracker := NewExpTracker(args)

	go tracker.Run()

	// Allow some time for the tracker to perform measurements.
	time.Sleep(250 * time.Millisecond)
	tracker.Stop()

	result := <-tracker.Ch
	if result.Err == nil {
		t.Log("Expected an error, but got nil")
		t.Fail()
	}
}

func TestExpTracker_CalculateStats(t *testing.T) {
	args := Args{
		ExpCrawler: &MockExpCrawler{},
		Clock:      newFakeClock(),
		Logger:     logger.NewLogger(),
	}
	tracker := NewExpTracker(args)

	firstMeasurement := Measurement{
		exp:        1000,
		percentage: 50,
		createdAt:  args.Clock.Now().Add(-time.Second * 100), // 1.6 mins
	}

	secondMeasurement := Measurement{
		exp:        1100,
		percentage: 60,
		createdAt:  args.Clock.Now(),
	}

	tracker.measurements = append(tracker.measurements, firstMeasurement, secondMeasurement)

	// ExpPerSecond = (1100 - 1000) / 100 = 1.0
	// PercentPerSecond = (60 - 60) / 100 = 0.1
	// TimeToLevelUpPerSecond = (100 - 60) / 0.1 = 400
	snaps.MatchJSON(t, tracker.getStats())
}
