package sysmetrics

import (
	"context"
	"time"

	"github.com/gopheramol/NetWatch/internal/battery"
	"github.com/gopheramol/NetWatch/internal/models"
	"github.com/gopheramol/NetWatch/internal/repository"
	"github.com/gopheramol/NetWatch/internal/utils"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/sensors"
	"go.uber.org/zap"
)

// Service collects system metrics (CPU, RAM, Disk, Temp, Battery) and persists them.
type Service interface {
	Collect(ctx context.Context) (*models.SystemMetrics, error)
	Latest(ctx context.Context) (*models.SystemMetrics, error)
	History(ctx context.Context, from, to time.Time, limit int) ([]models.SystemMetrics, error)
}

type service struct {
	repo          repository.SystemMetricsRepository
	batteryReader battery.Reader
	logger        *zap.Logger
}

// NewService builds a new SystemMetrics Service.
func NewService(repo repository.SystemMetricsRepository, batteryReader battery.Reader, logger *zap.Logger) Service {
	return &service{
		repo:          repo,
		batteryReader: batteryReader,
		logger:        logger,
	}
}

func (s *service) Collect(ctx context.Context) (*models.SystemMetrics, error) {
	metrics := &models.SystemMetrics{
		ID:        utils.NewID(),
		Timestamp: time.Now(),
	}

	// CPU Usage
	cpuPercents, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	if err == nil && len(cpuPercents) > 0 {
		metrics.CPUPercent = cpuPercents[0]
	}

	// Memory Usage
	vMem, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil && vMem != nil {
		metrics.RAMTotalMB = float64(vMem.Total) / (1024 * 1024)
		metrics.RAMUsedMB = float64(vMem.Used) / (1024 * 1024)
		metrics.RAMPercent = vMem.UsedPercent
	}

	// Disk Usage (root mount "/")
	diskUsage, err := disk.UsageWithContext(ctx, "/")
	if err == nil && diskUsage != nil {
		metrics.DiskTotalGB = float64(diskUsage.Total) / (1024 * 1024 * 1024)
		metrics.DiskUsedGB = float64(diskUsage.Used) / (1024 * 1024 * 1024)
		metrics.DiskPercent = diskUsage.UsedPercent
	}

	// CPU Temperature
	temps, err := sensors.TemperaturesWithContext(ctx)
	if err == nil && len(temps) > 0 {
		var maxTemp float64
		for _, t := range temps {
			if t.Temperature > maxTemp {
				maxTemp = t.Temperature
			}
		}
		metrics.CPUTempC = maxTemp
	}

	// Battery Reading
	if s.batteryReader != nil {
		if bReading, err := s.batteryReader.Read(ctx); err == nil && bReading.Present {
			metrics.BatteryPresent = true
			metrics.BatteryPercent = bReading.Percent
			metrics.BatteryCharging = bReading.Charging
		}
	}



	if err := s.repo.Save(ctx, metrics); err != nil {
		s.logger.Error("sysmetrics: failed to save metrics", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("sysmetrics: collected system metrics",
		zap.Float64("cpu_percent", metrics.CPUPercent),
		zap.Float64("ram_percent", metrics.RAMPercent),
		zap.Float64("disk_percent", metrics.DiskPercent),
		zap.Float64("cpu_temp_c", metrics.CPUTempC),
	)

	return metrics, nil
}

func (s *service) Latest(ctx context.Context) (*models.SystemMetrics, error) {
	return s.repo.Latest(ctx)
}

func (s *service) History(ctx context.Context, from, to time.Time, limit int) ([]models.SystemMetrics, error) {
	return s.repo.Range(ctx, from, to, limit)
}
