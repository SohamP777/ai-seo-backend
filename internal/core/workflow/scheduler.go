// pkg/workflow/scheduler.go
package workflow

import (
    "sync"
    "log"
    "io"
    

    "github.com/robfig/cron/v3"
    
)

// Scheduler handles scheduled workflow executions
type Scheduler struct {
    cron      *cron.Cron
    jobs      map[string]cron.EntryID
    workflows map[string]string // jobID -> workflowID
    mu        sync.RWMutex
    logger    *log.Logger
}

/// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
    return &Scheduler{
        cron:      cron.New(cron.WithSeconds()),
        jobs:      make(map[string]cron.EntryID),
        workflows: make(map[string]string),
        logger:    log.New(io.Discard, "", 0),
    }
}

// ScheduleWorkflow schedules a workflow to run on a cron schedule
func (s *Scheduler) ScheduleWorkflow(workflowID string, schedule string, input map[string]interface{}) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Check if already scheduled
    if _, exists := s.jobs[workflowID]; exists {
        return nil // Already scheduled
    }

    // Create job
    job := &ScheduledJob{
        WorkflowID: workflowID,
        Input:      input,
        scheduler:  s,
    }

    entryID, err := s.cron.AddJob(schedule, job)
    if err != nil {
        return err
    }

    s.jobs[workflowID] = entryID
    s.workflows[string(entryID)] = workflowID

    return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() {
    s.cron.Start()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
    s.cron.Stop()
}

// RemoveSchedule removes a scheduled workflow
func (s *Scheduler) RemoveSchedule(workflowID string) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if entryID, ok := s.jobs[workflowID]; ok {
        s.cron.Remove(entryID)
        delete(s.jobs, workflowID)
        delete(s.workflows, string(entryID))
    }
}

// ScheduledJob represents a scheduled workflow job
type ScheduledJob struct {
    WorkflowID string
    Input      map[string]interface{}
    scheduler  *Scheduler
}

// Run executes the scheduled job
func (j *ScheduledJob) Run() {
    j.scheduler.logger.Printf("Executing scheduled workflow workflow_id=%s", j.WorkflowID)
    // In real implementation, this would trigger workflow execution
}