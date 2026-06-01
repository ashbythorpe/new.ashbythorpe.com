package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3/log"
)

// implements fiber.Service
type DBCleanupService struct {
    db     *sql.DB
    state  string
	cancel context.CancelFunc
}

func NewDBCleanupService(db *sql.DB) *DBCleanupService {
	return &DBCleanupService{
		db: db,
		state: "initialized",
	}
}

func (s *DBCleanupService) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go s.runWorker(workerCtx)

	return nil
}

func (s *DBCleanupService) runWorker(ctx context.Context) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.cleanup(ctx); err != nil {
				logger := log.WithContext(ctx)
				logger.Error(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *DBCleanupService) cleanup(ctx context.Context) error {
	now := time.Now().Unix()
	logger := log.WithContext(ctx)

	res, err := s.db.Exec("DELETE FROM sessions WHERE expiry < ?", now)

	if err != nil {
		return err
	}

	if affected, err := res.RowsAffected(); err == nil && affected > 0 {
		logger.Infof("Cleaned up %d expired sessions", affected)
	}

	res, err = s.db.Exec("DELETE FROM verification_tokens WHERE expiry < ?", now)

	if err != nil {
		return err
	}

	if affected, err := res.RowsAffected(); err == nil && affected > 0 {
		logger.Infof("Cleaned up %d expired verification tokens", affected)
	}

	res, err = s.db.Exec("DELETE FROM password_reset_tokens WHERE expiry < ?", now)

	if err != nil {
		return err
	}

	if affected, err := res.RowsAffected(); err == nil && affected > 0 {
		logger.Infof("Cleaned up %d expired password reset tokens", affected)
	}

	return nil
}

func (s *DBCleanupService) String() string {
    return "Database Cleanup Service"
}

func (s *DBCleanupService) State(ctx context.Context) (string, error) {
    return s.state, nil
}

func (s *DBCleanupService) Terminate(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	s.state = "terminated"
	return nil
}
