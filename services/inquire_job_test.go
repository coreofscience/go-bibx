package services_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/coreofscience/go-bibx/clients/openalex"
	openalexmocks "github.com/coreofscience/go-bibx/internal/mocks/clients/openalex"
	"github.com/coreofscience/go-bibx/models"
	"github.com/coreofscience/go-bibx/repository"
	repositorytest "github.com/coreofscience/go-bibx/repository/test"
	"github.com/coreofscience/go-bibx/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubAnalysisService struct {
	inquireFn func(ctx context.Context, query string, limit int, opts services.InquireOptions) (*services.InquireResult, error)
}

func (s *stubAnalysisService) Inquire(ctx context.Context, query string, limit int, opts services.InquireOptions) (*services.InquireResult, error) {
	return s.inquireFn(ctx, query, limit, opts)
}

func (s *stubAnalysisService) StoreAnalysis(context.Context, uuid.UUID, *models.Analysis) error {
	return nil
}

func (s *stubAnalysisService) LoadAnalysis(context.Context, uuid.UUID) (*models.Analysis, error) {
	return nil, nil
}

func (s *stubAnalysisService) QueryCategory(context.Context, uuid.UUID, string, int) ([]*models.Result, error) {
	return nil, nil
}

func (s *stubAnalysisService) Search(context.Context, uuid.UUID, string, int) (*models.Analysis, error) {
	return nil, nil
}

func (s *stubAnalysisService) GetWorks(context.Context, uuid.UUID, []string) ([]*models.Work, []string, error) {
	return nil, nil, nil
}

func createTestCollection(t *testing.T, db *sql.DB, query string, limit int, name *string) *services.Collection {
	t.Helper()
	colSvc := services.NewCollectionService(db)
	col := &services.Collection{
		ID:          uuid.NewV7(),
		QueryHash:   services.ComputeQueryHash(query, limit),
		Query:       services.NormalizeQuery(query),
		ResultLimit: limit,
		Name:        name,
		WorkCount:   25,
	}
	require.NoError(t, colSvc.Create(t.Context(), col))
	saved, err := colSvc.FindByID(t.Context(), col.ID)
	require.NoError(t, err)
	return saved
}

func TestOpenAlexAnalysisService_InquireEmitsStages(t *testing.T) {
	t.Parallel()

	title1 := "Paper One"
	title2 := "Paper Two"
	w1 := &openalex.Work{
		ID:              "https://openalex.org/W1",
		Title:           &title1,
		IDs:             map[string]string{"openalex": "https://openalex.org/W1"},
		ReferencedWorks: []string{"https://openalex.org/W2"},
	}
	w2 := &openalex.Work{
		ID:    "https://openalex.org/W2",
		Title: &title2,
		IDs:   map[string]string{"openalex": "https://openalex.org/W2"},
	}

	oaClient := openalexmocks.NewMockClient(t)
	oaClient.EXPECT().
		SearchWorks(mock.Anything, "graph theory", 10).
		Return([]*openalex.Work{w1, w2}, nil).
		Once()
	oaClient.EXPECT().
		ListWorksByIDs(mock.Anything, mock.Anything).
		Return([]*openalex.Work{w2}, nil).
		Maybe()

	db := repositorytest.NewDB(t)
	anaSvc := services.NewAnalysisService(db, oaClient)

	var stages []services.InquireStage
	res, err := anaSvc.Inquire(t.Context(), "graph theory", 10, services.InquireOptions{
		OnStage: func(stage services.InquireStage) {
			stages = append(stages, stage)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, []services.InquireStage{
		services.StageFetchingSeeds,
		services.StageEnriching,
		services.StageScoring,
		services.StageStoring,
	}, stages)
}

func TestSQLiteInquireJobService_CacheHitFastPath(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	name := "cached-streams"
	existing := createTestCollection(t, db, "stream processing", 200, &name)

	var calls atomic.Int32
	anaSvc := &stubAnalysisService{
		inquireFn: func(context.Context, string, int, services.InquireOptions) (*services.InquireResult, error) {
			calls.Add(1)
			return nil, errors.New("should not be called on cache hit")
		},
	}

	jobSvc := services.NewInquireJobService(db, anaSvc)
	defer jobSvc.Close()

	job, err := jobSvc.Start(t.Context(), "  Stream   Processing ", 200, services.InquireOptions{})
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusCompleted, job.Status)
	assert.Equal(t, services.StageDone, job.Stage)
	assert.False(t, job.IsNew)
	require.NotNil(t, job.Collection)
	assert.Equal(t, existing.ID, job.Collection.ID)
	assert.Zero(t, calls.Load())
}

func TestSQLiteInquireJobService_StartWaitAndStages(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	release := make(chan struct{})
	inFetching := make(chan struct{})

	anaSvc := &stubAnalysisService{
		inquireFn: func(ctx context.Context, query string, limit int, opts services.InquireOptions) (*services.InquireResult, error) {
			if opts.OnStage != nil {
				opts.OnStage(services.StageFetchingSeeds)
			}
			close(inFetching)
			<-release
			if opts.OnStage != nil {
				opts.OnStage(services.StageEnriching)
				opts.OnStage(services.StageScoring)
				opts.OnStage(services.StageStoring)
			}
			col := createTestCollection(t, db, query, limit, nil)
			return &services.InquireResult{
				Collection: col,
				IsNew:      true,
				SeedCount:  175,
			}, nil
		},
	}

	jobSvc := services.NewInquireJobService(db, anaSvc)
	defer jobSvc.Close()

	var observedStages []services.InquireStage
	var stagesMu sync.Mutex

	started, err := jobSvc.Start(t.Context(), "quantum computing", 200, services.InquireOptions{
		OnStage: func(stage services.InquireStage) {
			stagesMu.Lock()
			observedStages = append(observedStages, stage)
			stagesMu.Unlock()
		},
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil(), started.ID)

	<-inFetching

	// Short wait expires while job is still running.
	mid, err := jobSvc.Wait(t.Context(), started.ID, 20*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusRunning, mid.Status)
	assert.Equal(t, services.StageFetchingSeeds, mid.Stage)

	// Allow job to finish; Wait wakes up immediately.
	close(release)
	done, err := jobSvc.Wait(t.Context(), started.ID, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusCompleted, done.Status)
	assert.Equal(t, services.StageDone, done.Stage)
	assert.True(t, done.IsNew)
	assert.Equal(t, 175, done.SeedCount)
	require.NotNil(t, done.Collection)
	assert.Equal(t, "quantum computing", done.Collection.Query)

	stagesMu.Lock()
	assert.Equal(t, []services.InquireStage{
		services.StageFetchingSeeds,
		services.StageEnriching,
		services.StageScoring,
		services.StageStoring,
	}, observedStages)
	stagesMu.Unlock()
}

func TestSQLiteInquireJobService_DeduplicatesConcurrentStarts(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	release := make(chan struct{})
	var calls atomic.Int32

	anaSvc := &stubAnalysisService{
		inquireFn: func(ctx context.Context, query string, limit int, opts services.InquireOptions) (*services.InquireResult, error) {
			calls.Add(1)
			<-release
			col := createTestCollection(t, db, query, limit, nil)
			return &services.InquireResult{Collection: col, IsNew: true, SeedCount: 50}, nil
		},
	}

	jobSvc := services.NewInquireJobService(db, anaSvc)
	defer jobSvc.Close()

	job1, err := jobSvc.Start(t.Context(), "apache kafka", 100, services.InquireOptions{})
	require.NoError(t, err)

	job2, err := jobSvc.Start(t.Context(), "  Apache   Kafka ", 100, services.InquireOptions{})
	require.NoError(t, err)
	assert.Equal(t, job1.ID, job2.ID)

	close(release)
	done, err := jobSvc.Wait(t.Context(), job1.ID, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusCompleted, done.Status)
	assert.Equal(t, int32(1), calls.Load())
}

func TestSQLiteInquireJobService_PreflightNameConflict(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	taken := "my-topic"
	createTestCollection(t, db, "other topic", 100, &taken)

	anaSvc := &stubAnalysisService{
		inquireFn: func(context.Context, string, int, services.InquireOptions) (*services.InquireResult, error) {
			return nil, errors.New("should not run")
		},
	}

	jobSvc := services.NewInquireJobService(db, anaSvc)
	defer jobSvc.Close()

	_, err := jobSvc.Start(t.Context(), "new topic", 100, services.InquireOptions{Name: "my-topic"})
	require.ErrorIs(t, err, services.ErrCollectionExists)
}

func TestSQLiteInquireJobService_RecordsFailure(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	anaSvc := &stubAnalysisService{
		inquireFn: func(context.Context, string, int, services.InquireOptions) (*services.InquireResult, error) {
			return nil, errors.New("openalex rate limit exceeded")
		},
	}

	jobSvc := services.NewInquireJobService(db, anaSvc)
	defer jobSvc.Close()

	started, err := jobSvc.Start(t.Context(), "failing query", 100, services.InquireOptions{})
	require.NoError(t, err)

	failed, err := jobSvc.Wait(t.Context(), started.ID, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusFailed, failed.Status)
	assert.Contains(t, failed.Error, "openalex rate limit exceeded")
}

func TestSQLiteInquireJobService_CrossProcessPollingFallback(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	release := make(chan struct{})

	anaSvc := &stubAnalysisService{
		inquireFn: func(ctx context.Context, query string, limit int, opts services.InquireOptions) (*services.InquireResult, error) {
			<-release
			col := createTestCollection(t, db, query, limit, nil)
			return &services.InquireResult{Collection: col, IsNew: true, SeedCount: 80}, nil
		},
	}

	runnerSvc := services.NewInquireJobService(db, anaSvc)
	defer runnerSvc.Close()

	// Second service instance has no local waiter channel for runnerSvc's jobs.
	readerSvc := services.NewInquireJobService(db, anaSvc, services.WithPollInterval(10*time.Millisecond))
	defer readerSvc.Close()

	started, err := runnerSvc.Start(t.Context(), "distributed systems", 100, services.InquireOptions{})
	require.NoError(t, err)

	go func() {
		time.Sleep(30 * time.Millisecond)
		close(release)
	}()

	done, err := readerSvc.Wait(t.Context(), started.ID, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusCompleted, done.Status)
	require.NotNil(t, done.Collection)
}

func TestSQLiteInquireJobService_CloseCancelsInFlight(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	startedCh := make(chan struct{})

	anaSvc := &stubAnalysisService{
		inquireFn: func(ctx context.Context, _ string, _ int, _ services.InquireOptions) (*services.InquireResult, error) {
			close(startedCh)
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	jobSvc := services.NewInquireJobService(db, anaSvc)

	job, err := jobSvc.Start(t.Context(), "long query", 100, services.InquireOptions{})
	require.NoError(t, err)

	<-startedCh
	jobSvc.Close()

	after, err := jobSvc.Get(t.Context(), job.ID)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusFailed, after.Status)
	assert.Contains(t, after.Error, "context canceled")

	_, err = jobSvc.Start(t.Context(), "another query", 100, services.InquireOptions{})
	require.ErrorIs(t, err, services.ErrJobServiceClosed)
}

func TestSQLiteInquireJobService_RecoverStaleAndGetNotFound(t *testing.T) {
	t.Parallel()

	db := repositorytest.NewDB(t)
	q := repository.New(db)

	hash := services.ComputeQueryHash("orphaned query", 100)
	orphanID := uuid.NewV7()
	require.NoError(t, q.CreateInquireJob(t.Context(), repository.CreateInquireJobParams{
		ID:          orphanID,
		QueryHash:   hash[:],
		Query:       "orphaned query",
		ResultLimit: 100,
		Status:      string(services.JobStatusRunning),
		Stage:       string(services.StageEnriching),
	}))

	jobSvc := services.NewInquireJobService(db, &stubAnalysisService{})
	defer jobSvc.Close()

	require.NoError(t, jobSvc.RecoverStale(t.Context(), 0))

	recovered, err := jobSvc.Get(t.Context(), orphanID)
	require.NoError(t, err)
	assert.Equal(t, services.JobStatusFailed, recovered.Status)
	assert.Contains(t, recovered.Error, "interrupted")

	_, err = jobSvc.Get(t.Context(), uuid.NewV7())
	require.ErrorIs(t, err, services.ErrJobNotFound)
}
