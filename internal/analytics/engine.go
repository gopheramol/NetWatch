// Package analytics incrementally maintains daily and monthly summary
// buckets as new connectivity checks, speed tests, and outages occur, so
// that reporting endpoints never need to scan the raw time-series data.
package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/repository"
	"go.uber.org/zap"
)

const (
	dateFormat  = "2006-01-02"
	monthFormat = "2006-01"
)

// Engine keeps daily_stats and monthly_stats up to date in response to new
// data, and serves pre-aggregated reads back to callers.
type Engine interface {
	RecordCheck(ctx context.Context, check models.ConnectivityCheck) error
	RecordSpeedTest(ctx context.Context, result models.SpeedTestResult) error
	RecordOutageClosed(ctx context.Context, outage models.Outage) error

	GetDaily(ctx context.Context, date string) (*models.DailyStats, error)
	GetDailyRange(ctx context.Context, from, to time.Time) ([]models.DailyStats, error)
	GetMonthly(ctx context.Context, month string) (*models.MonthlyStats, error)
	GetMonthlyRange(ctx context.Context, from, to time.Time) ([]models.MonthlyStats, error)
}

type engine struct {
	daily   repository.DailyStatsRepository
	monthly repository.MonthlyStatsRepository
	logger  *zap.Logger
}

// NewEngine builds the default analytics Engine.
func NewEngine(daily repository.DailyStatsRepository, monthly repository.MonthlyStatsRepository, logger *zap.Logger) Engine {
	return &engine{daily: daily, monthly: monthly, logger: logger}
}

// loadDaily returns the stats for date, or a fresh zero-value record if none exists yet.
func (e *engine) loadDaily(ctx context.Context, date string) (*models.DailyStats, error) {
	stats, err := e.daily.Get(ctx, date)
	if err != nil {
		return &models.DailyStats{Date: date}, nil
	}
	return stats, nil
}

func (e *engine) loadMonthly(ctx context.Context, month string) (*models.MonthlyStats, error) {
	stats, err := e.monthly.Get(ctx, month)
	if err != nil {
		return &models.MonthlyStats{Month: month}, nil
	}
	return stats, nil
}

// RecordCheck folds a single connectivity check into its day's and month's
// running latency average and availability figures.
func (e *engine) RecordCheck(ctx context.Context, check models.ConnectivityCheck) error {
	date := check.Timestamp.Format(dateFormat)
	month := check.Timestamp.Format(monthFormat)

	daily, err := e.loadDaily(ctx, date)
	if err != nil {
		return err
	}
	daily.CheckCount++
	daily.LatencySum += check.LatencyMs
	daily.AvgLatencyMs = daily.LatencySum / float64(daily.CheckCount)
	recomputeDailyAvailability(daily, check.Timestamp)
	if err := e.daily.Save(ctx, daily); err != nil {
		return fmt.Errorf("saving daily stats: %w", err)
	}

	monthly, err := e.loadMonthly(ctx, month)
	if err != nil {
		return err
	}
	monthly.CheckCount++
	monthly.LatencySum += check.LatencyMs
	monthly.AvgLatencyMs = monthly.LatencySum / float64(monthly.CheckCount)
	recomputeMonthlyAvailability(monthly, check.Timestamp)
	if err := e.monthly.Save(ctx, monthly); err != nil {
		return fmt.Errorf("saving monthly stats: %w", err)
	}

	return nil
}

// RecordSpeedTest folds a speed test result into its day's and month's
// running speed average, min, and max.
func (e *engine) RecordSpeedTest(ctx context.Context, result models.SpeedTestResult) error {
	date := result.Timestamp.Format(dateFormat)
	month := result.Timestamp.Format(monthFormat)

	daily, err := e.loadDaily(ctx, date)
	if err != nil {
		return err
	}
	if daily.SpeedTestCount == 0 {
		daily.MinSpeedMbps = result.DownloadMbps
		daily.MaxSpeedMbps = result.DownloadMbps
	} else {
		if result.DownloadMbps < daily.MinSpeedMbps {
			daily.MinSpeedMbps = result.DownloadMbps
		}
		if result.DownloadMbps > daily.MaxSpeedMbps {
			daily.MaxSpeedMbps = result.DownloadMbps
		}
	}
	daily.SpeedTestCount++
	daily.SpeedSum += result.DownloadMbps
	daily.AvgSpeedMbps = daily.SpeedSum / float64(daily.SpeedTestCount)
	if err := e.daily.Save(ctx, daily); err != nil {
		return fmt.Errorf("saving daily stats: %w", err)
	}

	monthly, err := e.loadMonthly(ctx, month)
	if err != nil {
		return err
	}
	monthly.SpeedTestCount++
	monthly.SpeedSum += result.DownloadMbps
	monthly.AvgSpeedMbps = monthly.SpeedSum / float64(monthly.SpeedTestCount)
	if err := e.monthly.Save(ctx, monthly); err != nil {
		return fmt.Errorf("saving monthly stats: %w", err)
	}

	return nil
}

// daySegment is the portion of an outage that overlaps a single calendar day.
type daySegment struct {
	date    string
	month   string
	seconds float64
}

func splitByDay(start, end time.Time) []daySegment {
	var segments []daySegment
	cursor := start
	for cursor.Before(end) {
		dayEnd := time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, cursor.Location()).AddDate(0, 0, 1)
		segEnd := dayEnd
		if end.Before(segEnd) {
			segEnd = end
		}
		segments = append(segments, daySegment{
			date:    cursor.Format(dateFormat),
			month:   cursor.Format(monthFormat),
			seconds: segEnd.Sub(cursor).Seconds(),
		})
		cursor = segEnd
	}
	return segments
}

// RecordOutageClosed folds a resolved outage into every day and month it
// overlapped, distributing downtime seconds proportionally and updating
// outage counts / longest / average outage duration for the outage's month.
func (e *engine) RecordOutageClosed(ctx context.Context, outage models.Outage) error {
	if outage.EndTime == nil {
		return fmt.Errorf("cannot record an unresolved outage")
	}

	segments := splitByDay(outage.StartTime, *outage.EndTime)
	touchedMonths := map[string]bool{}
	startMonth := outage.StartTime.Format(monthFormat)

	for i, seg := range segments {
		daily, err := e.loadDaily(ctx, seg.date)
		if err != nil {
			return err
		}
		daily.DowntimeSeconds += seg.seconds
		if i == 0 {
			daily.OutageCount++
		}
		recomputeDailyAvailability(daily, outage.StartTime)
		if err := e.daily.Save(ctx, daily); err != nil {
			return fmt.Errorf("saving daily stats: %w", err)
		}

		monthly, err := e.loadMonthly(ctx, seg.month)
		if err != nil {
			return err
		}
		monthly.DowntimeSeconds += seg.seconds
		if !touchedMonths[seg.month] {
			touchedMonths[seg.month] = true
			if seg.month == startMonth {
				monthly.OutageCount++
				durationSec := outage.Duration.Seconds()
				if durationSec > monthly.LongestOutageSec {
					monthly.LongestOutageSec = durationSec
				}
				monthly.AvgOutageSec = ((monthly.AvgOutageSec * float64(monthly.OutageCount-1)) + durationSec) / float64(monthly.OutageCount)
			}
		}
		recomputeMonthlyAvailability(monthly, outage.StartTime)
		if err := e.monthly.Save(ctx, monthly); err != nil {
			return fmt.Errorf("saving monthly stats: %w", err)
		}
	}

	return nil
}

// recomputeDailyAvailability derives availability_pct from downtime_seconds
// against the elapsed portion of the day as of `now`, so partial (in
// progress) days report a meaningful figure rather than assuming 24h elapsed.
func recomputeDailyAvailability(stats *models.DailyStats, now time.Time) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	elapsed := now.Sub(dayStart).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	stats.AvailabilityPct = clampPct(100 * (1 - stats.DowntimeSeconds/elapsed))
}

func recomputeMonthlyAvailability(stats *models.MonthlyStats, now time.Time) {
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	elapsed := now.Sub(monthStart).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	stats.AvailabilityPct = clampPct(100 * (1 - stats.DowntimeSeconds/elapsed))
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func (e *engine) GetDaily(ctx context.Context, date string) (*models.DailyStats, error) {
	return e.loadDaily(ctx, date)
}

func (e *engine) GetDailyRange(ctx context.Context, from, to time.Time) ([]models.DailyStats, error) {
	return e.daily.Range(ctx, from.Format(dateFormat), to.Format(dateFormat))
}

func (e *engine) GetMonthly(ctx context.Context, month string) (*models.MonthlyStats, error) {
	return e.loadMonthly(ctx, month)
}

func (e *engine) GetMonthlyRange(ctx context.Context, from, to time.Time) ([]models.MonthlyStats, error) {
	return e.monthly.Range(ctx, from.Format(monthFormat), to.Format(monthFormat))
}
