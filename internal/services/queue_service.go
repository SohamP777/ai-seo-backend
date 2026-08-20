package services

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
)

type JobStatus string

const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
)

type Job struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Data      map[string]interface{} `json:"data"`
    Status    JobStatus              `json:"status"`
    CreatedAt time.Time              `json:"created_at"`
    StartedAt time.Time               `json:"started_at,omitempty"`
    CompletedAt time.Time             `json:"completed_at,omitempty"`
    Error     string                 `json:"error,omitempty"`
    Result    interface{}            `json:"result,omitempty"`
}

type JobHandler func(job Job) (interface{}, error)

type QueueService struct {
    jobs      map[string]Job
    handlers  map[string]JobHandler
    mu        sync.RWMutex
    workerPool chan struct{}
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

// NewQueueService creates a new queue service
func NewQueueService(workers int) *QueueService {
    ctx, cancel := context.WithCancel(context.Background())
    
    q := &QueueService{
        jobs:       make(map[string]Job),
        handlers:   make(map[string]JobHandler),
        workerPool: make(chan struct{}, workers),
        ctx:        ctx,
        cancel:     cancel,
    }

    // Start worker pool
    for i := 0; i < workers; i++ {
        q.workerPool <- struct{}{}
    }

    return q
}

// RegisterHandler registers a handler for a job type
func (q *QueueService) RegisterHandler(jobType string, handler JobHandler) {
    q.mu.Lock()
    defer q.mu.Unlock()
    q.handlers[jobType] = handler
}

// Enqueue adds a job to the queue
func (q *QueueService) Enqueue(jobType string, data map[string]interface{}) (string, error) {
    q.mu.Lock()
    defer q.mu.Unlock()

    // Check if handler exists
    if _, ok := q.handlers[jobType]; !ok {
        return "", fmt.Errorf("no handler registered for job type: %s", jobType)
    }

    // Generate job ID
    jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

    // Create job
    job := Job{
        ID:        jobID,
        Type:      jobType,
        Data:      data,
        Status:    JobStatusPending,
        CreatedAt: time.Now(),
    }

    q.jobs[jobID] = job

    // Process job asynchronously
    go q.processJob(jobID)

    return jobID, nil
}

// EnqueueBatch adds multiple jobs to the queue
func (q *QueueService) EnqueueBatch(jobs []struct {
    Type string
    Data map[string]interface{}
}) []string {
    var jobIDs []string
    
    for _, j := range jobs {
        jobID, err := q.Enqueue(j.Type, j.Data)
        if err == nil {
            jobIDs = append(jobIDs, jobID)
        }
    }
    
    return jobIDs
}

// GetJob returns a job by ID
func (q *QueueService) GetJob(jobID string) (Job, error) {
    q.mu.RLock()
    defer q.mu.RUnlock()

    job, ok := q.jobs[jobID]
    if !ok {
        return Job{}, fmt.Errorf("job not found: %s", jobID)
    }

    return job, nil
}

// GetJobs returns jobs by status
func (q *QueueService) GetJobs(status JobStatus, limit int) []Job {
    q.mu.RLock()
    defer q.mu.RUnlock()

    var jobs []Job
    for _, job := range q.jobs {
        if job.Status == status {
            jobs = append(jobs, job)
            if len(jobs) >= limit {
                break
            }
        }
    }
    return jobs
}

// CancelJob cancels a pending job
func (q *QueueService) CancelJob(jobID string) error {
    q.mu.Lock()
    defer q.mu.Unlock()

    job, ok := q.jobs[jobID]
    if !ok {
        return fmt.Errorf("job not found: %s", jobID)
    }

    if job.Status != JobStatusPending {
        return fmt.Errorf("cannot cancel job with status: %s", job.Status)
    }

    job.Status = JobStatusFailed
    job.Error = "cancelled by user"
    job.CompletedAt = time.Now()
    q.jobs[jobID] = job

    return nil
}

// RetryFailedJob retries a failed job
func (q *QueueService) RetryFailedJob(jobID string) error {
    q.mu.Lock()
    defer q.mu.Unlock()

    job, ok := q.jobs[jobID]
    if !ok {
        return fmt.Errorf("job not found: %s", jobID)
    }

    if job.Status != JobStatusFailed {
        return fmt.Errorf("cannot retry job with status: %s", job.Status)
    }

    job.Status = JobStatusPending
    job.Error = ""
    job.StartedAt = time.Time{}
    job.CompletedAt = time.Time{}
    job.Result = nil
    q.jobs[jobID] = job

    go q.processJob(jobID)

    return nil
}

// GetQueueStats returns queue statistics
func (q *QueueService) GetQueueStats() map[string]interface{} {
    q.mu.RLock()
    defer q.mu.RUnlock()

    stats := map[string]interface{}{
        "total":      len(q.jobs),
        "pending":    0,
        "running":    0,
        "completed":  0,
        "failed":     0,
        "workers":    cap(q.workerPool),
        "available_workers": len(q.workerPool),
    }

    for _, job := range q.jobs {
        switch job.Status {
        case JobStatusPending:
            stats["pending"] = stats["pending"].(int) + 1
        case JobStatusRunning:
            stats["running"] = stats["running"].(int) + 1
        case JobStatusCompleted:
            stats["completed"] = stats["completed"].(int) + 1
        case JobStatusFailed:
            stats["failed"] = stats["failed"].(int) + 1
        }
    }

    return stats
}

// Shutdown gracefully shuts down the queue
func (q *QueueService) Shutdown() {
    q.cancel()
    q.wg.Wait()
}

// processJob processes a job
func (q *QueueService) processJob(jobID string) {
    // Wait for available worker
    <-q.workerPool
    defer func() { q.workerPool <- struct{}{} }()

    q.wg.Add(1)
    defer q.wg.Done()

    // Get job
    q.mu.Lock()
    job, ok := q.jobs[jobID]
    if !ok {
        q.mu.Unlock()
        return
    }

    // Check if job is still pending
    if job.Status != JobStatusPending {
        q.mu.Unlock()
        return
    }

    // Update job status
    job.Status = JobStatusRunning
    job.StartedAt = time.Now()
    q.jobs[jobID] = job
    q.mu.Unlock()

    // Get handler
    q.mu.RLock()
    handler, ok := q.handlers[job.Type]
    q.mu.RUnlock()

    if !ok {
        q.updateJobStatus(jobID, JobStatusFailed, nil, fmt.Errorf("no handler found"))
        return
    }

    // Process job with timeout
    resultChan := make(chan interface{})
    errorChan := make(chan error)

    go func() {
        result, err := handler(job)
        if err != nil {
            errorChan <- err
        } else {
            resultChan <- result
        }
    }()

    select {
    case result := <-resultChan:
        q.updateJobStatus(jobID, JobStatusCompleted, result, nil)
    case err := <-errorChan:
        q.updateJobStatus(jobID, JobStatusFailed, nil, err)
    case <-time.After(5 * time.Minute):
        q.updateJobStatus(jobID, JobStatusFailed, nil, fmt.Errorf("job timeout after 5 minutes"))
    case <-q.ctx.Done():
        q.updateJobStatus(jobID, JobStatusFailed, nil, fmt.Errorf("job cancelled during shutdown"))
    }
}

// updateJobStatus updates job status
func (q *QueueService) updateJobStatus(jobID string, status JobStatus, result interface{}, err error) {
    q.mu.Lock()
    defer q.mu.Unlock()

    job, ok := q.jobs[jobID]
    if !ok {
        return
    }

    job.Status = status
    job.CompletedAt = time.Now()
    job.Result = result
    
    if err != nil {
        job.Error = err.Error()
    }

    q.jobs[jobID] = job
}

// Predefined job handlers

// KeywordAnalysisJob handles keyword analysis jobs
func KeywordAnalysisJob(job Job) (interface{}, error) {
    // Extract job data
    keyword, ok := job.Data["keyword"].(string)
    if !ok {
        return nil, fmt.Errorf("keyword not provided")
    }

    url, _ := job.Data["url"].(string)

    // Simulate analysis
    time.Sleep(2 * time.Second)

    result := map[string]interface{}{
        "keyword":    keyword,
        "volume":     1000,
        "difficulty": 45.5,
        "cpc":        2.5,
        "analyzed_at": time.Now(),
    }

    if url != "" {
        result["url"] = url
        result["density"] = 1.8
    }

    return result, nil
}

// CompetitorAnalysisJob handles competitor analysis jobs
func CompetitorAnalysisJob(job Job) (interface{}, error) {
    yourURL, ok := job.Data["your_url"].(string)
    if !ok {
        return nil, fmt.Errorf("your_url not provided")
    }

    competitors, _ := job.Data["competitors"].([]interface{})

    // Simulate analysis
    time.Sleep(3 * time.Second)

    var compList []string
    for _, c := range competitors {
        if str, ok := c.(string); ok {
            compList = append(compList, str)
        }
    }

    result := map[string]interface{}{
        "your_url":      yourURL,
        "competitors":   compList,
        "gap_keywords":  []string{"seo tools", "keyword research", "rank tracking"},
        "opportunities": 12,
        "analyzed_at":   time.Now(),
    }

    return result, nil
}

// ReportGenerationJob handles report generation jobs
func ReportGenerationJob(job Job) (interface{}, error) {
    userID, ok := job.Data["user_id"].(float64)
    if !ok {
        return nil, fmt.Errorf("user_id not provided")
    }

    reportType, _ := job.Data["type"].(string)

    // Simulate report generation
    time.Sleep(5 * time.Second)

    result := map[string]interface{}{
        "user_id":     int(userID),
        "type":        reportType,
        "report_url":  fmt.Sprintf("/reports/%d/%s.pdf", int(userID), reportType),
        "generated_at": time.Now(),
    }

    return result, nil
}

// EmailJob handles email sending jobs
func EmailJob(job Job) (interface{}, error) {
    to, ok := job.Data["to"].(string)
    if !ok {
        return nil, fmt.Errorf("recipient not provided")
    }

    subject, _ := job.Data["subject"].(string)
    template, _ := job.Data["template"].(string)

    // Simulate email sending
    log.Printf("Sending email to %s: %s", to, subject)
    time.Sleep(1 * time.Second)

    result := map[string]interface{}{
        "to":        to,
        "subject":   subject,
        "template":  template,
        "sent_at":   time.Now(),
    }

    return result, nil
}