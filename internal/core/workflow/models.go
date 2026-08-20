package workflow

import "time"

// Workflow represents a workflow definition
type Workflow struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Version     string                 `json:"version"`
    Nodes       map[string]*WorkflowNode `json:"nodes"`
    Edges       []*WorkflowEdge        `json:"edges"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    Status      string                 `json:"status"`
    Tags        []string               `json:"tags"`
    Timeout     time.Duration          `json:"timeout"`
    MaxRetries  int                    `json:"max_retries"`
}

// WorkflowNode represents a node in the workflow DAG
type WorkflowNode struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"` // task, condition, start, end, human, loop
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    TaskType    string                 `json:"task_type"` // crawl, scan, keyword, report, notify
    Config      map[string]interface{} `json:"config"`
    Inputs      []string               `json:"inputs"`
    Outputs     []string               `json:"outputs"`
    RetryCount  int                    `json:"retry_count"`
    Timeout     time.Duration          `json:"timeout"`
    Next        []string               `json:"next,omitempty"`
    Position    struct {
        X float64 `json:"x"`
        Y float64 `json:"y"`
    } `json:"position,omitempty"`
}

// WorkflowEdge represents an edge between nodes
type WorkflowEdge struct {
    ID        string `json:"id"`
    From      string `json:"from"`
    To        string `json:"to"`
    Condition string `json:"condition,omitempty"`
    Type      string `json:"type,omitempty"` // success, failure, default
}

// WorkflowInstance represents a running workflow instance
type WorkflowInstance struct {
    ID                 string                 `json:"id"`
    WorkflowID         string                 `json:"workflow_id"`
    WorkflowTemplateID string                 `json:"workflow_template_id,omitempty"`
    UserID             string                 `json:"user_id"`
    Status             string                 `json:"status"` // running, paused, completed, failed
    Data               map[string]interface{} `json:"data,omitempty"`
    Input              map[string]interface{} `json:"input,omitempty"`
    Output             map[string]interface{} `json:"output,omitempty"`
    CreatedBy          string                 `json:"created_by"`
    StartedAt          time.Time              `json:"started_at"`
    EndedAt            *time.Time             `json:"ended_at,omitempty"`
    Priority           int                    `json:"priority,omitempty"`
    Tags               map[string]string      `json:"tags,omitempty"`
}

// NodeExecution represents execution of a single node
type NodeExecution struct {
    ID                 string                 `json:"id"`
    WorkflowInstanceID string                 `json:"instance_id"`
    NodeID             string                 `json:"node_id"`
    Status             string                 `json:"status"`
    Input              map[string]interface{} `json:"input,omitempty"`
    Output             map[string]interface{} `json:"output,omitempty"`
    Error              string                 `json:"error,omitempty"`
    StartedAt          time.Time              `json:"started_at"`
    EndedAt            *time.Time             `json:"ended_at,omitempty"`
    RetryAttempt       int                    `json:"retry_attempt,omitempty"`
    Duration           time.Duration          `json:"duration,omitempty"`
}

// WorkflowTemplate represents a reusable workflow template
type WorkflowTemplate struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Category    string                 `json:"category"`
    Description string                 `json:"description"`
    Nodes       []WorkflowNode         `json:"nodes"`
    Edges       []WorkflowEdge         `json:"edges"`
    Variables   []string               `json:"variables,omitempty"`
    Status      string                 `json:"status"`
    CreatedBy   string                 `json:"created_by"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    EstimatedDuration time.Duration    `json:"estimated_duration,omitempty"`
}

// WorkflowMetrics represents metrics for a workflow
type WorkflowMetrics struct {
    WorkflowID       string                 `json:"workflow_id"`
    TotalExecutions  int                    `json:"total_runs"`
    SuccessfulRuns   int                    `json:"successful_runs"`
    FailedRuns       int                    `json:"failed_runs"`
    SuccessRate      float64                `json:"success_rate"`
    AvgDuration      time.Duration          `json:"avg_duration"`
    AvgWaitTime      time.Duration          `json:"avg_wait_time,omitempty"`
    LastRun          *time.Time             `json:"last_run,omitempty"`
    NodeMetrics      map[string]NodeMetrics `json:"node_metrics"`
    FailurePoints    []string               `json:"failure_points,omitempty"`
    Bottlenecks      []string               `json:"bottlenecks,omitempty"`
}

// NodeMetrics represents metrics for a specific node
type NodeMetrics struct {
    NodeID           string        `json:"node_id"`
    TotalExecutions  int           `json:"total_executions"`
    Failures         int           `json:"failures,omitempty"`
    AvgDuration      time.Duration `json:"avg_duration"`
    ErrorRate        float64       `json:"error_rate"`
    RetryRate        float64       `json:"retry_rate,omitempty"`
}

// HumanTask represents a task requiring human intervention
type HumanTask struct {
    ID           string                 `json:"id"`
    InstanceID   string                 `json:"instance_id"`
    NodeID       string                 `json:"node_id"`
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    Input        map[string]interface{} `json:"input"`
    AssignedTo   string                 `json:"assigned_to"`
    Status       string                 `json:"status"` // pending, completed, rejected
    CreatedAt    time.Time              `json:"created_at"`
    CompletedAt  *time.Time             `json:"completed_at,omitempty"`
    Output       map[string]interface{} `json:"output,omitempty"`
    Comments     string                 `json:"comments,omitempty"`
    Priority     int                    `json:"priority,omitempty"`
    Deadline     *time.Time             `json:"deadline,omitempty"`
}