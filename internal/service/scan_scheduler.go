package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"ulas-service/internal/repository"
	"ulas-service/models"
)

// ScanScheduler periodically triggers scheduled scans.
type ScanScheduler struct {
	scheduleRepo repository.ScanScheduleRepository
	scanService  ScanService
	ticker       *time.Ticker
	stop         chan struct{}
}

// NewScanScheduler creates a new scan scheduler.
func NewScanScheduler(scheduleRepo repository.ScanScheduleRepository, scanService ScanService) *ScanScheduler {
	return &ScanScheduler{
		scheduleRepo: scheduleRepo,
		scanService:  scanService,
		stop:         make(chan struct{}),
	}
}

// Start begins the scheduler loop.
func (s *ScanScheduler) Start() {
	if s.ticker != nil {
		return
	}
	s.ticker = time.NewTicker(1 * time.Minute)
	go s.loop()
}

// Stop halts the scheduler loop.
func (s *ScanScheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
		close(s.stop)
		s.ticker = nil
	}
}

func (s *ScanScheduler) loop() {
	for {
		select {
		case <-s.ticker.C:
			s.runDueSchedules()
		case <-s.stop:
			return
		}
	}
}

func (s *ScanScheduler) runDueSchedules() {
	ctx := context.Background()
	now := time.Now().UTC()

	schedules, err := s.scheduleRepo.ListEnabledDue(ctx, now)
	if err != nil {
		slog.Error("failed to list due scan schedules", "error", err)
		return
	}

	for i := range schedules {
		schedule := &schedules[i]
		startedAt := time.Now().UTC()

		slog.Info("triggering scheduled scan",
			"schedule_id", schedule.ID,
			"name", schedule.Name,
			"frequency", schedule.Frequency,
			"limit", schedule.Limit,
		)

		job, err := s.scanService.InitiateScan(ctx, schedule.Limit, "scheduler")
		if err != nil {
			if errors.Is(err, ErrScanAlreadyRunning) {
				slog.Info("scheduled scan skipped because another scan is already running",
					"schedule_id", schedule.ID,
					"name", schedule.Name,
				)
				continue
			}

			errMsg := err.Error()
			slog.Error("scheduled scan failed",
				"schedule_id", schedule.ID,
				"name", schedule.Name,
				"error", errMsg,
			)

			run := &models.ScanScheduleRun{
				ID:           uuid.New(),
				ScheduleID:   schedule.ID,
				Status:       models.ScanScheduleRunStatusFailed,
				ErrorMessage: errMsg,
				StartedAt:    startedAt,
				CreatedAt:    time.Now().UTC(),
			}
			if createErr := s.scheduleRepo.CreateRun(ctx, run); createErr != nil {
				slog.Error("failed to record failed scan schedule run",
					"schedule_id", schedule.ID,
					"error", createErr,
				)
			}

			// Advance the schedule so it does not retry immediately.
			s.advanceSchedule(ctx, schedule, startedAt)
			continue
		}

		run := &models.ScanScheduleRun{
			ID:         uuid.New(),
			ScheduleID: schedule.ID,
			ScanJobID:  &job.ID,
			Status:     models.ScanScheduleRunStatusSuccess,
			StartedAt:  startedAt,
			CreatedAt:  time.Now().UTC(),
		}
		if createErr := s.scheduleRepo.CreateRun(ctx, run); createErr != nil {
			slog.Error("failed to record successful scan schedule run",
				"schedule_id", schedule.ID,
				"error", createErr,
			)
		}

		s.advanceSchedule(ctx, schedule, startedAt)
	}
}

func (s *ScanScheduler) advanceSchedule(ctx context.Context, schedule *models.ScanSchedule, lastRun time.Time) {
	nextRun := schedule.ComputeNextRun(lastRun)
	schedule.LastRunAt = &lastRun
	schedule.NextRunAt = &nextRun
	schedule.UpdatedAt = time.Now().UTC()

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		slog.Error("failed to update scan schedule after run",
			"schedule_id", schedule.ID,
			"name", schedule.Name,
			"error", err,
		)
	}
}

// ScanSchedulerStatus returns a simple status string for health checks.
func (s *ScanScheduler) ScanSchedulerStatus() string {
	if s.ticker != nil {
		return "running"
	}
	return "stopped"
}

