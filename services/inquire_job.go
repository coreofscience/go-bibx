package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/repository"
)

// JobStatus represents the lifecycle state of an asynchronous inquiry job.
type JobStatus string

// Lifecycle statuses for an asynchronous inquiry job.
const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// IsTerminal reports whether the job has reached a final state.
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed
}

var (
	// ErrJobNotFound is returned when an inquiry job ID does not exist.
	ErrJobNotFound = errors.New("inquire job not found")

	// ErrJobServiceClosed is returned when Start is called after Close.
	ErrJobServiceClosed = errors.New("inquire job service is closed")
)

const (
	defaultMaxConcurrentJobs = 2
	defaultJobPollInterval   = 500 * time.Millisecond
	jobFinalizeTimeout       = 5 * time.Second
)

// InquireJob represents an asynchronous bibliographic inquiry job.
type InquireJob struct {
	ID           uuid.UUID    `json:"id"`
	Query        string       `json:"query"`
	ResultLimit  int          `json:"result_limit"`
	Name         *string      `json:"name,omitempty"`
	Status       JobStatus    `json:"status"`
	Stage        InquireStage `json:"stage"`
	CollectionID *uuid.UUID   `json:"collection_id,omitempty"`
	Collection   *Collection  `json:"collection,omitempty"`
	IsNew        bool         `json:"is_new"`
	SeedCount    int          `json:"seed_count"`
	Error        string       `json:"error,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// InquireJobService manages asynchronous inquiry jobs backed by durable state.
type InquireJobService interface {
	Start(ctx context.Context, query string, limit int, opts InquireOptions) (*InquireJob, error)
	Get(ctx context.Context, id uuid.UUID) (*InquireJob, error)
	Wait(ctx context.Context, id uuid.UUID, timeout time.Duration) (*InquireJob, error)
}

// InquireJobOption configures [SQLiteInquireJobService].
type InquireJobOption func(*inquireJobConfig)

type inquireJobConfig struct {
	parentCtx     context.Context
	maxConcurrent int
	pollInterval  time.Duration
}

// WithJobContext sets the parent lifecycle context for background jobs.
// Defaults to [context.Background].
func WithJobContext(ctx context.Context) InquireJobOption {
	return func(c *inquireJobConfig) {
		if ctx != nil {
			c.parentCtx = ctx
		}
	}
}

// WithMaxConcurrentJobs sets the maximum number of concurrent inquire jobs.
// Defaults to 2.
func WithMaxConcurrentJobs(n int) InquireJobOption {
	return func(c *inquireJobConfig) {
		if n > 0 {
			c.maxConcurrent = n
		}
	}
}

// WithPollInterval sets the SQLite fallback poll interval used by Wait when
// the job is not being executed by this process instance. Defaults to 500ms.
func WithPollInterval(d time.Duration) InquireJobOption {
	return func(c *inquireJobConfig) {
		if d > 0 {
			c.pollInterval = d
		}
	}
}

// SQLiteInquireJobService implements [InquireJobService] using SQLite for
// durable job state and bounded in-process goroutines for execution.
type SQLiteInquireJobService struct {
	queries         *repository.Queries
	analysisService AnalysisService
	baseCtx         context.Context
	cancel          context.CancelFunc
	sem             chan struct{}
	pollInterval    time.Duration
	wg              sync.WaitGroup
	mu              sync.Mutex
	waiters         map[uuid.UUID]chan struct{}
	closed          bool
}

// NewInquireJobService creates a new [SQLiteInquireJobService].
func NewInquireJobService(
	db repository.DBTX,
	analysisService AnalysisService,
	opts ...InquireJobOption,
) *SQLiteInquireJobService {
	cfg := inquireJobConfig{
		parentCtx:     context.Background(),
		maxConcurrent: defaultMaxConcurrentJobs,
		pollInterval:  defaultJobPollInterval,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	baseCtx, cancel := context.WithCancel(cfg.parentCtx)
	return &SQLiteInquireJobService{
		queries:         repository.New(db),
		analysisService: analysisService,
		baseCtx:         baseCtx,
		cancel:          cancel,
		sem:             make(chan struct{}, cfg.maxConcurrent),
		pollInterval:    cfg.pollInterval,
		waiters:         make(map[uuid.UUID]chan struct{}),
	}
}

// Close cancels all in-flight background jobs and waits for them to finish.
// Call Close before closing the underlying database connection.
func (s *SQLiteInquireJobService) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	s.cancel()
	s.wg.Wait()
}

// RecoverStale marks orphaned pending or running jobs older than maxAge as
// failed. Pass maxAge <= 0 on single-instance startup to fail all leftover
// active jobs immediately.
func (s *SQLiteInquireJobService) RecoverStale(ctx context.Context, maxAge time.Duration) error {
	var cutoff time.Time
	if maxAge <= 0 {
		cutoff = time.Now().UTC().Add(time.Second)
	} else {
		cutoff = time.Now().UTC().Add(-maxAge)
	}
	if err := s.queries.FailStaleInquireJobs(ctx, repository.FailStaleInquireJobsParams{
		Error:     sql.NullString{String: "job interrupted before completion", Valid: true},
		UpdatedAt: sql.NullTime{Time: cutoff, Valid: true},
	}); err != nil {
		return fmt.Errorf("failed to recover stale inquire jobs: %w", err)
	}
	return nil
}

// Start begins an asynchronous inquiry job, or returns immediately when a
// cached collection or an identical in-flight job already exists.
func (s *SQLiteInquireJobService) Start(
	ctx context.Context,
	query string,
	limit int,
	opts InquireOptions,
) (*InquireJob, error) {
	normQuery := NormalizeQuery(query)
	queryHash := ComputeQueryHash(query, limit)

	existing, outdated, err := s.findExistingCollection(ctx, normQuery, limit, queryHash)
	if err != nil {
		return nil, err
	}

	// Fast path: return cached collection immediately without spawning a job.
	if existing != nil && !outdated && !opts.Force {
		colID := existing.ID
		return &InquireJob{
			Query:        existing.Query,
			ResultLimit:  existing.ResultLimit,
			Name:         existing.Name,
			Status:       JobStatusCompleted,
			Stage:        StageDone,
			CollectionID: &colID,
			Collection:   existing,
			IsNew:        false,
			CreatedAt:    existing.CreatedAt,
			UpdatedAt:    existing.UpdatedAt,
		}, nil
	}

	// Pre-flight name conflict check so invalid names fail synchronously.
	var nameNull sql.NullString
	if trimmed := strings.TrimSpace(opts.Name); trimmed != "" {
		if err := s.checkNameAvailable(ctx, trimmed, existing); err != nil {
			return nil, err
		}
		nameNull = sql.NullString{String: trimmed, Valid: true}
	}

	// Deduplicate against any active job for the same query hash.
	if active, err := s.queries.GetActiveInquireJobByHash(ctx, queryHash[:]); err == nil {
		return toInquireJob(active), nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check active inquire job: %w", err)
	}

	jobID := uuid.NewV7()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrJobServiceClosed
	}

	err = s.queries.CreateInquireJob(ctx, repository.CreateInquireJobParams{
		ID:          jobID,
		QueryHash:   queryHash[:],
		Query:       normQuery,
		ResultLimit: int64(limit),
		Name:        nameNull,
		Force:       opts.Force,
		Status:      string(JobStatusPending),
		Stage:       string(StageQueued),
	})
	if err != nil {
		s.mu.Unlock()
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			active, lookupErr := s.queries.GetActiveInquireJobByHash(ctx, queryHash[:])
			if lookupErr == nil {
				return toInquireJob(active), nil
			}
		}
		return nil, fmt.Errorf("failed to create inquire job: %w", err)
	}

	s.waiters[jobID] = make(chan struct{})
	s.wg.Add(1)
	s.mu.Unlock()

	go s.runJob(jobID, query, limit, opts)

	row, err := s.queries.GetInquireJobByID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to load created inquire job: %w", err)
	}
	return toInquireJob(row), nil
}

// Get returns the current state of an inquiry job, populating Collection when
// the job has completed.
func (s *SQLiteInquireJobService) Get(ctx context.Context, id uuid.UUID) (*InquireJob, error) {
	row, err := s.queries.GetInquireJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get inquire job: %w", err)
	}

	job := toInquireJob(row)
	if job.Status == JobStatusCompleted && job.CollectionID != nil {
		colRow, err := s.queries.GetCollectionByID(ctx, *job.CollectionID)
		if err == nil {
			job.Collection = toCollection(colRow)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to load collection for inquire job: %w", err)
		}
	}
	return job, nil
}

// Wait blocks until the job reaches a terminal state, timeout elapses, or ctx
// is canceled, then returns the latest job state.
func (s *SQLiteInquireJobService) Wait(
	ctx context.Context,
	id uuid.UUID,
	timeout time.Duration,
) (*InquireJob, error) {
	job, err := s.Get(ctx, id)
	if err != nil || job.Status.IsTerminal() || timeout <= 0 {
		return job, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	if ch := s.waiter(id); ch != nil {
		select {
		case <-ch:
			return s.Get(ctx, id)
		case <-timer.C:
			return s.Get(ctx, id)
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for inquire job canceled: %w", ctx.Err())
		}
	}

	// Fallback for jobs executing in another process or worker.
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			job, err = s.Get(ctx, id)
			if err != nil || job.Status.IsTerminal() {
				return job, err
			}
		case <-timer.C:
			return s.Get(ctx, id)
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for inquire job canceled: %w", ctx.Err())
		}
	}
}

func (s *SQLiteInquireJobService) runJob(
	jobID uuid.UUID,
	query string,
	limit int,
	opts InquireOptions,
) {
	defer s.wg.Done()
	defer s.notifyDone(jobID)

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-s.baseCtx.Done():
		s.failJob(jobID, s.baseCtx.Err())
		return
	}

	s.updateStage(jobID, JobStatusRunning, StageFetchingSeeds)

	userOnStage := opts.OnStage
	runOpts := opts
	runOpts.OnStage = func(stage InquireStage) {
		s.updateStage(jobID, JobStatusRunning, stage)
		notifyStage(userOnStage, stage)
	}

	res, err := s.analysisService.Inquire(s.baseCtx, query, limit, runOpts)
	if err != nil {
		s.failJob(jobID, err)
		return
	}

	var colID *uuid.UUID
	if res != nil && res.Collection != nil {
		id := res.Collection.ID
		colID = &id
	}

	finCtx, cancel := context.WithTimeout(context.WithoutCancel(s.baseCtx), jobFinalizeTimeout)
	defer cancel()

	if err := s.queries.CompleteInquireJob(finCtx, repository.CompleteInquireJobParams{
		CollectionID: colID,
		IsNew:        res != nil && res.IsNew,
		SeedCount:    int64(seedCount(res)),
		ID:           jobID,
	}); err != nil {
		slog.ErrorContext(finCtx, "failed to mark inquire job completed", "job_id", jobID, "error", err)
	}
}

func seedCount(res *InquireResult) int {
	if res == nil {
		return 0
	}
	return res.SeedCount
}

func (s *SQLiteInquireJobService) updateStage(jobID uuid.UUID, status JobStatus, stage InquireStage) {
	if err := s.queries.UpdateInquireJobStage(s.baseCtx, repository.UpdateInquireJobStageParams{
		Status: string(status),
		Stage:  string(stage),
		ID:     jobID,
	}); err != nil {
		slog.WarnContext(s.baseCtx, "failed to update inquire job stage",
			"job_id", jobID,
			"stage", stage,
			"error", err,
		)
	}
}

func (s *SQLiteInquireJobService) failJob(jobID uuid.UUID, jobErr error) {
	finCtx, cancel := context.WithTimeout(context.WithoutCancel(s.baseCtx), jobFinalizeTimeout)
	defer cancel()

	msg := "unknown error"
	if jobErr != nil {
		msg = jobErr.Error()
	}

	if err := s.queries.FailInquireJob(finCtx, repository.FailInquireJobParams{
		Error: sql.NullString{String: msg, Valid: true},
		ID:    jobID,
	}); err != nil {
		slog.ErrorContext(finCtx, "failed to mark inquire job failed", "job_id", jobID, "error", err)
	}
}

func (s *SQLiteInquireJobService) waiter(id uuid.UUID) chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.waiters[id]
}

func (s *SQLiteInquireJobService) notifyDone(id uuid.UUID) {
	s.mu.Lock()
	ch, ok := s.waiters[id]
	if ok {
		delete(s.waiters, id)
	}
	s.mu.Unlock()
	if ok {
		close(ch)
	}
}

func (s *SQLiteInquireJobService) findExistingCollection(
	ctx context.Context,
	normQuery string,
	limit int,
	queryHash [32]byte,
) (*Collection, bool, error) {
	current, err := s.queries.GetCollectionByHash(ctx, queryHash[:])
	if err == nil {
		return toCollection(current), false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("failed to check existing collection: %w", err)
	}

	older, err := s.queries.GetCollectionByQuery(ctx, repository.GetCollectionByQueryParams{
		Query:       normQuery,
		ResultLimit: int64(limit),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to check outdated collection: %w", err)
	}
	return toCollection(older), true, nil
}

func (s *SQLiteInquireJobService) checkNameAvailable(ctx context.Context, name string, target *Collection) error {
	holder, err := s.queries.GetCollectionByName(ctx, sql.NullString{String: name, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check collection name: %w", err)
	}
	if target != nil && holder.ID == target.ID {
		return nil
	}
	return fmt.Errorf("name %q is used by the collection for query %q: %w", name, holder.Query, ErrCollectionExists)
}

func toInquireJob(r repository.InquireJob) *InquireJob {
	var namePtr *string
	if r.Name.Valid {
		namePtr = &r.Name.String
	}

	var errStr string
	if r.Error.Valid {
		errStr = r.Error.String
	}

	var createdAt, updatedAt time.Time
	if r.CreatedAt.Valid {
		createdAt = r.CreatedAt.Time.UTC()
	}
	if r.UpdatedAt.Valid {
		updatedAt = r.UpdatedAt.Time.UTC()
	}

	return &InquireJob{
		ID:           r.ID,
		Query:        r.Query,
		ResultLimit:  int(r.ResultLimit),
		Name:         namePtr,
		Status:       JobStatus(r.Status),
		Stage:        InquireStage(r.Stage),
		CollectionID: r.CollectionID,
		IsNew:        r.IsNew,
		SeedCount:    int(r.SeedCount),
		Error:        errStr,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
