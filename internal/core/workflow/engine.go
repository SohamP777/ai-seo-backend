package workflow

import (
    "context"
    "fmt"
    "sync"
    "time"
    "log" 
    "io"

    "github.com/google/uuid"
)

// TaskExecutor defines the interface for executing tasks
type TaskExecutor interface {
    Execute(ctx context.Context, node *WorkflowNode, data map[string]interface{}) (map[string]interface{}, error)
    Validate(node *WorkflowNode) error
}

// Engine is the core workflow execution engine
type Engine struct {
    mu              sync.RWMutex
    workflows       map[string]*Workflow
    instances       map[string]*WorkflowInstance
    nodeExecutions  map[string]*NodeExecution
    templates       map[string]*WorkflowTemplate
    metrics         map[string]*WorkflowMetrics
    logger          *log.Logger
    taskExecutors   map[string]TaskExecutor
    maxConcurrent   int
    semaphore       chan struct{}
}

func (e *Engine) StartEngine() {
    e.logger.Printf("Starting workflow engine with %d workers", e.maxConcurrent)
}

func (e *Engine) StopEngine() {
    e.logger.Printf("Stopping workflow engine")
}

// NewEngine creates a new workflow engine
func NewEngine(logger *log.Logger, maxConcurrent int) *Engine {
    if logger == nil {
        logger = log.New(io.Discard, "", 0)
    }
    return &Engine{
        workflows:      make(map[string]*Workflow),
        instances:      make(map[string]*WorkflowInstance),
        nodeExecutions: make(map[string]*NodeExecution),
        templates:      make(map[string]*WorkflowTemplate),
        metrics:        make(map[string]*WorkflowMetrics),
        logger:         logger,
        taskExecutors:  make(map[string]TaskExecutor),
        maxConcurrent:  maxConcurrent,
        semaphore:      make(chan struct{}, maxConcurrent),
    }
}

// RegisterTaskExecutor registers a task executor for a specific task type
func (e *Engine) RegisterTaskExecutor(taskType string, executor TaskExecutor) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.taskExecutors[taskType] = executor
    e.logger.Printf("Registered task executor task_type=%s", taskType)
}

// CreateWorkflow creates a new workflow
func (e *Engine) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    if workflow.ID == "" {
        workflow.ID = uuid.New().String()
    }
    workflow.CreatedAt = time.Now()
    workflow.UpdatedAt = time.Now()
    workflow.Status = "active"

    // Validate workflow
    if err := e.validateWorkflow(workflow); err != nil {
        return fmt.Errorf("workflow validation failed: %w", err)
    }

    e.workflows[workflow.ID] = workflow
    e.logger.Printf("Workflow created id=%s name=%s", workflow.ID, workflow.Name)
    
    return nil
}

// validateWorkflow validates a workflow definition
func (e *Engine) validateWorkflow(workflow *Workflow) error {
    if len(workflow.Nodes) == 0 {
        return fmt.Errorf("workflow must have at least one node")
    }

    // Check for start node
    hasStart := false
    hasEnd := false
    
    for _, node := range workflow.Nodes {
        if node.Type == "start" {
            hasStart = true
        }
        if node.Type == "end" {
            hasEnd = true
        }
    }

    if !hasStart {
        return fmt.Errorf("workflow must have a start node")
    }
    if !hasEnd {
        return fmt.Errorf("workflow must have an end node")
    }

    // Validate edges
    for _, edge := range workflow.Edges {
        if _, ok := workflow.Nodes[edge.From]; !ok {
            return fmt.Errorf("edge from node %s does not exist", edge.From)
        }
        if _, ok := workflow.Nodes[edge.To]; !ok {
            return fmt.Errorf("edge to node %s does not exist", edge.To)
        }
    }

    return nil
}

// StartWorkflow starts a new workflow instance
func (e *Engine) StartWorkflow(ctx context.Context, workflowID string, input map[string]interface{}, userID string) (*WorkflowInstance, error) {
    e.mu.RLock()
    workflow, ok := e.workflows[workflowID]
    e.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("workflow not found: %s", workflowID)
    }

    instance := &WorkflowInstance{
        ID:         uuid.New().String(),
        WorkflowID: workflowID,
        UserID:     userID,
        Status:     "running",
        Data:       make(map[string]interface{}),
        Input:      input,
        CreatedBy:  userID,
        StartedAt:  time.Now(),
        Priority:   1,
        Tags:       make(map[string]string),
    }

    // Copy input to data
    for k, v := range input {
        instance.Data[k] = v
    }

    e.mu.Lock()
    e.instances[instance.ID] = instance
    
    // Initialize metrics
    if _, exists := e.metrics[workflowID]; !exists {
        e.metrics[workflowID] = &WorkflowMetrics{
            WorkflowID: workflowID,
        }
    }
    e.mu.Unlock()

    // Find start node
    var startNode *WorkflowNode
    for _, node := range workflow.Nodes {
        if node.Type == "start" {
            startNode = node
            break
        }
    }

    if startNode == nil {
        instance.Status = "failed"
        instance.EndedAt = ptrTime(time.Now())
        return instance, fmt.Errorf("no start node found")
    }

    // Execute workflow asynchronously
    go e.executeWorkflow(ctx, instance, startNode)

    e.logger.Printf("Workflow started instance_id=%s workflow_id=%s user_id=%s", 
        instance.ID, workflowID, userID)

    return instance, nil
}

// executeWorkflow executes a workflow instance
func (e *Engine) executeWorkflow(ctx context.Context, instance *WorkflowInstance, startNode *WorkflowNode) {
    e.semaphore <- struct{}{}
    defer func() { <-e.semaphore }()

    e.mu.RLock()
    workflow := e.workflows[instance.WorkflowID]
    e.mu.RUnlock()

    if workflow == nil {
        e.logger.Printf("Workflow not found during execution workflow_id=%s", instance.WorkflowID)
        instance.Status = "failed"
        instance.EndedAt = ptrTime(time.Now())
        return
    }

    // Create execution context
    execCtx := &ExecutionContext{
        Instance: instance,
        Workflow: workflow,
        Visited:  make(map[string]bool),
        Engine:   e,
    }

    // Start execution from start node
    if err := e.executeNode(ctx, execCtx, startNode); err != nil {
        instance.Status = "failed"
        instance.EndedAt = ptrTime(time.Now())
        e.logger.Printf("Workflow execution failed instance_id=%s error=%v", instance.ID, err)
        return
    }

    instance.Status = "completed"
    instance.EndedAt = ptrTime(time.Now())
    instance.Output = instance.Data
    
    e.logger.Printf("Workflow completed instance_id=%s duration=%v", 
        instance.ID, instance.EndedAt.Sub(instance.StartedAt))

    // Update metrics
    e.updateWorkflowMetrics(instance, true)
}

// executeNode executes a single workflow node
func (e *Engine) executeNode(ctx context.Context, execCtx *ExecutionContext, node *WorkflowNode) error {
    execCtx.mu.Lock()
    if execCtx.Visited[node.ID] {
        execCtx.mu.Unlock()
        return nil // Prevent cycles
    }
    execCtx.Visited[node.ID] = true
    execCtx.mu.Unlock()

    // Create node execution record
    execution := &NodeExecution{
        ID:                 uuid.New().String(),
        WorkflowInstanceID: execCtx.Instance.ID,
        NodeID:             node.ID,
        Status:             "running",
        StartedAt:          time.Now(),
        RetryAttempt:       0,
        Input:              make(map[string]interface{}),
    }

    // Prepare input data
    for _, key := range node.Inputs {
        if val, ok := execCtx.Instance.Data[key]; ok {
            execution.Input[key] = val
        }
    }

    e.mu.Lock()
    e.nodeExecutions[execution.ID] = execution
    e.mu.Unlock()

    e.logger.Printf("Executing node instance_id=%s node_id=%s node_type=%s", 
        execCtx.Instance.ID, node.ID, node.Type)

    // Execute based on node type
    var err error
    var output map[string]interface{}

    switch node.Type {
    case "start":
        output = make(map[string]interface{})
    case "end":
        output = make(map[string]interface{})
    case "task":
        output, err = e.executeTask(ctx, node, execCtx)
    default:
        output = make(map[string]interface{})
    }

    execution.EndedAt = ptrTime(time.Now())
    execution.Duration = execution.EndedAt.Sub(execution.StartedAt)

    if err != nil {
        execution.Status = "failed"
        execution.Error = err.Error()
        
        e.logger.Printf("Node execution failed node_id=%s error=%v", node.ID, err)
        
        return err
    }

    execution.Status = "completed"
    execution.Output = output
    
    // Update instance data with node output
    if output != nil {
        for k, v := range output {
            execCtx.Instance.Data[k] = v
        }
    }

    e.logger.Printf("Node completed node_id=%s duration=%v", node.ID, execution.Duration)

    // Find next nodes
    return e.executeNextNodes(ctx, execCtx, node)
}

// executeTask executes a task node
func (e *Engine) executeTask(ctx context.Context, node *WorkflowNode, execCtx *ExecutionContext) (map[string]interface{}, error) {
    e.mu.RLock()
    executor, ok := e.taskExecutors[node.TaskType]
    e.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("no executor for task type: %s", node.TaskType)
    }

    // Execute with retries
    var lastErr error
    var output map[string]interface{}
    
    for attempt := 0; attempt <= node.RetryCount; attempt++ {
        if attempt > 0 {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(time.Second * time.Duration(attempt*2)):
                // Exponential backoff
            }
        }

        output, lastErr = executor.Execute(ctx, node, execCtx.Instance.Data)
        if lastErr == nil {
            return output, nil
        }
        
        e.logger.Printf("Task execution failed, retrying node=%s attempt=%d error=%v",
            node.ID, attempt, lastErr)
    }

    return nil, fmt.Errorf("task execution failed after %d retries: %w", node.RetryCount, lastErr)
}

// executeNextNodes finds and executes the next nodes
func (e *Engine) executeNextNodes(ctx context.Context, execCtx *ExecutionContext, currentNode *WorkflowNode) error {
    workflow := execCtx.Workflow
    
    // Find outgoing edges
    var nextNodes []string
    for _, edge := range workflow.Edges {
        if edge.From == currentNode.ID {
            nextNodes = append(nextNodes, edge.To)
        }
    }

    // Also check Next field for backward compatibility
    if len(nextNodes) == 0 && len(currentNode.Next) > 0 {
        nextNodes = currentNode.Next
    }

    if len(nextNodes) == 0 {
        return nil // End of path
    }

    // Execute next nodes sequentially
    for _, nodeID := range nextNodes {
        nextNode := workflow.Nodes[nodeID]
        if nextNode == nil {
            e.logger.Printf("Next node not found from=%s to=%s", currentNode.ID, nodeID)
            continue
        }

        if err := e.executeNode(ctx, execCtx, nextNode); err != nil {
            return err
        }
    }

    return nil
}

// updateWorkflowMetrics updates overall workflow metrics
func (e *Engine) updateWorkflowMetrics(instance *WorkflowInstance, success bool) {
    e.mu.Lock()
    defer e.mu.Unlock()

    metrics, ok := e.metrics[instance.WorkflowID]
    if !ok {
        return
    }

    metrics.TotalExecutions++
    if success {
        // Update successful runs count if field exists
        // Note: Your WorkflowMetrics struct in models.go is truncated
        // You may need to add these fields
    }

    now := time.Now()
    metrics.LastRun = &now
}

// GetWorkflowInstance retrieves a workflow instance
func (e *Engine) GetWorkflowInstance(instanceID string) (*WorkflowInstance, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    instance, ok := e.instances[instanceID]
    if !ok {
        return nil, fmt.Errorf("instance not found: %s", instanceID)
    }

    return instance, nil
}

// GetNodeExecutions retrieves node executions for an instance
func (e *Engine) GetNodeExecutions(instanceID string) ([]*NodeExecution, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    var executions []*NodeExecution
    for _, exec := range e.nodeExecutions {
        if exec.WorkflowInstanceID == instanceID {
            executions = append(executions, exec)
        }
    }

    return executions, nil
}

// GetWorkflowMetrics retrieves metrics for a workflow
func (e *Engine) GetWorkflowMetrics(workflowID string) (*WorkflowMetrics, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    metrics, ok := e.metrics[workflowID]
    if !ok {
        return nil, fmt.Errorf("metrics not found for workflow: %s", workflowID)
    }

    return metrics, nil
}

// GetWorkflowTemplates returns all templates
func (e *Engine) GetWorkflowTemplates() ([]*WorkflowTemplate, error) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    templates := make([]*WorkflowTemplate, 0, len(e.templates))
    for _, template := range e.templates {
        templates = append(templates, template)
    }

    return templates, nil
}

// CreateTemplate creates a template from an existing workflow
// CreateTemplate creates a template from an existing workflow
func (e *Engine) CreateTemplate(workflowID string, name string, category string, createdBy string) (*WorkflowTemplate, error) {
    e.mu.RLock()
    workflow, ok := e.workflows[workflowID]
    e.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("workflow not found: %s", workflowID)
    }

    nodes := make([]WorkflowNode, 0, len(workflow.Nodes))
    for _, node := range workflow.Nodes {
        nodes = append(nodes, *node)
    }

    // Convert []*WorkflowEdge to []WorkflowEdge
    edges := make([]WorkflowEdge, 0, len(workflow.Edges))
    for _, edge := range workflow.Edges {
        edges = append(edges, *edge)
    }

    template := &WorkflowTemplate{
        ID:          uuid.New().String(),
        Name:        name,
        Category:    category,
        Description: workflow.Description,
        Nodes:       nodes,
        Edges:       edges,  // Now using converted edges
        Variables:   []string{},
        Status:      "active",
        CreatedBy:   createdBy,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    e.mu.Lock()
    e.templates[template.ID] = template
    e.mu.Unlock()

    e.logger.Printf("Template created template_id=%s name=%s", template.ID, name)

    return template, nil
}

// GetVisualization generates a simple visualization
func (e *Engine) GetVisualization(workflowID string) (string, error) {
    e.mu.RLock()
    workflow, ok := e.workflows[workflowID]
    e.mu.RUnlock()

    if !ok {
        return "", fmt.Errorf("workflow not found: %s", workflowID)
    }

    dot := "digraph Workflow {\n"
    dot += "  rankdir=LR;\n"
    dot += "  node [shape=box, style=rounded];\n\n"

    // Add nodes
    for id, node := range workflow.Nodes {
        shape := "box"
        switch node.Type {
        case "start":
            shape = "ellipse"
        case "end":
            shape = "ellipse"
        }
        dot += fmt.Sprintf("  %s [label=\"%s\", shape=%s];\n", id, node.Name, shape)
    }

    dot += "\n"

    // Add edges
    for _, edge := range workflow.Edges {
        dot += fmt.Sprintf("  %s -> %s;\n", edge.From, edge.To)
    }

    dot += "}\n"
    return dot, nil
}

// ptrTime returns a pointer to a time.Time
func ptrTime(t time.Time) *time.Time {
    return &t
}

// ExecutionContext holds the context for workflow execution
type ExecutionContext struct {
    Instance *WorkflowInstance
    Workflow *Workflow
    Visited  map[string]bool
    Engine   *Engine
    mu       sync.RWMutex
}