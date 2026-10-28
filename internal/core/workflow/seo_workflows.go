// pkg/workflow/seo_workflows.go
package workflow

import (
    "time"
)

// CreateContentPublishingWorkflow creates a content publishing workflow
func CreateContentPublishingWorkflow() *Workflow {
    nodes := make(map[string]*WorkflowNode)
    edges := []*WorkflowEdge{}

    // Keyword Research Node
    nodes["keyword_research"] = &WorkflowNode{
        ID:          "keyword_research",
        Type:        "task",
        Name:        "Keyword Research",
        Description: "Research keywords for content",
        TaskType:    "keyword_research",
        Inputs:      []string{"topic"},
        Outputs:     []string{"primary_keyword", "keywords", "search_intent"},
        RetryCount:  3,
        Timeout:     5 * time.Minute,
    }

    // Content Brief Node
    nodes["content_brief"] = &WorkflowNode{
        ID:          "content_brief",
        Type:        "human",
        Name:        "Content Brief Creation",
        Description: "Create content brief based on keywords",
        TaskType:    "content_brief",
        Inputs:      []string{"primary_keyword", "keywords", "search_intent"},
        Outputs:     []string{"content_brief"},
        Config: map[string]interface{}{
            "assigned_to": "content_strategist",
        },
    }

    // Writer Assignment Node
    nodes["writer_assignment"] = &WorkflowNode{
        ID:          "writer_assignment",
        Type:        "task",
        Name:        "Writer Assignment",
        Description: "Assign writer based on topic",
        TaskType:    "assignment",
        Inputs:      []string{"content_brief"},
        Outputs:     []string{"writer", "deadline"},
    }

    // Draft Review Node
    nodes["draft_review"] = &WorkflowNode{
        ID:          "draft_review",
        Type:        "human",
        Name:        "Draft Review",
        Description: "Review initial draft",
        TaskType:    "review",
        Inputs:      []string{"draft_content", "content_brief"},
        Outputs:     []string{"review_comments", "revision_required"},
        Config: map[string]interface{}{
            "assigned_to": "editor",
        },
    }

    // SEO Check Node
    nodes["seo_check"] = &WorkflowNode{
        ID:          "seo_check",
        Type:        "task",
        Name:        "SEO Check",
        Description: "Check content for SEO optimization",
        TaskType:    "content_optimizer",
        Inputs:      []string{"draft_content", "primary_keyword"},
        Outputs:     []string{"seo_score", "suggestions", "optimized_content"},
    }

    // Internal Link Addition Node
    nodes["link_addition"] = &WorkflowNode{
        ID:          "link_addition",
        Type:        "task",
        Name:        "Internal Link Addition",
        Description: "Add internal links to content",
        TaskType:    "link_analyzer",
        Inputs:      []string{"optimized_content"},
        Outputs:     []string{"linked_content", "internal_links_added"},
    }

    // Image Optimization Node
    nodes["image_optimization"] = &WorkflowNode{
        ID:          "image_optimization",
        Type:        "task",
        Name:        "Image Optimization",
        Description: "Optimize images for SEO",
        TaskType:    "image_optimizer",
        Inputs:      []string{"images"},
        Outputs:     []string{"optimized_images", "alt_texts"},
    }

    // Final Approval Node
    nodes["final_approval"] = &WorkflowNode{
        ID:          "final_approval",
        Type:        "human",
        Name:        "Final Approval",
        Description: "Final approval before publishing",
        TaskType:    "approval",
        Inputs:      []string{"linked_content", "optimized_images"},
        Outputs:     []string{"approved", "approver_comments"},
        Config: map[string]interface{}{
            "assigned_to": "content_manager",
        },
    }

    // Publish Node
    nodes["publish"] = &WorkflowNode{
        ID:          "publish",
        Type:        "task",
        Name:        "Publish Content",
        Description: "Publish content to CMS",
        TaskType:    "publisher",
        Inputs:      []string{"approved", "linked_content"},
        Outputs:     []string{"published_url", "publish_date"},
    }

    // Indexing Request Node
    nodes["indexing_request"] = &WorkflowNode{
        ID:          "indexing_request",
        Type:        "task",
        Name:        "Request Indexing",
        Description: "Request search engine indexing",
        TaskType:    "indexing",
        Inputs:      []string{"published_url"},
        Outputs:     []string{"indexing_status"},
    }

    // Rank Tracking Setup Node
    nodes["rank_tracking"] = &WorkflowNode{
        ID:          "rank_tracking",
        Type:        "task",
        Name:        "Setup Rank Tracking",
        Description: "Setup rank tracking for published content",
        TaskType:    "rank_tracker",
        Inputs:      []string{"published_url", "primary_keyword"},
        Outputs:     []string{"tracking_id", "initial_rank"},
    }

    // Add Start and End nodes
    nodes["start"] = &WorkflowNode{
        ID:   "start",
        Type: "start",
        Name: "Start",
    }

    nodes["end"] = &WorkflowNode{
        ID:   "end",
        Type: "end",
        Name: "End",
    }

    // Define edges
    edges = append(edges,
        &WorkflowEdge{From: "start", To: "keyword_research"},
        &WorkflowEdge{From: "keyword_research", To: "content_brief"},
        &WorkflowEdge{From: "content_brief", To: "writer_assignment"},
        &WorkflowEdge{From: "writer_assignment", To: "draft_review"},
        
        // Conditional edge for revision loop
        &WorkflowEdge{From: "draft_review", To: "seo_check", Condition: "data.revision_required == false"},
        &WorkflowEdge{From: "draft_review", To: "writer_assignment", Condition: "data.revision_required == true", Type: "failure"},
        
        &WorkflowEdge{From: "seo_check", To: "link_addition"},
        &WorkflowEdge{From: "link_addition", To: "image_optimization"},
        &WorkflowEdge{From: "image_optimization", To: "final_approval"},
        &WorkflowEdge{From: "final_approval", To: "publish", Condition: "data.approved == true"},
        &WorkflowEdge{From: "final_approval", To: "draft_review", Condition: "data.approved == false", Type: "failure"},
        &WorkflowEdge{From: "publish", To: "indexing_request"},
        &WorkflowEdge{From: "indexing_request", To: "rank_tracking"},
        &WorkflowEdge{From: "rank_tracking", To: "end"},
    )

    return &Workflow{
        ID:          "content_publishing",
        Name:        "Content Publishing Workflow",
        Description: "Complete content creation and publishing workflow",
        Version:     "1.0.0",
        Nodes:       nodes,
        Edges:       edges,
        CreatedAt:   time.Now(),
        Status:      "active",
        MaxRetries:  3,
        Timeout:     24 * time.Hour,
    }
}

// CreateSiteMigrationWorkflow creates a site migration workflow
func CreateSiteMigrationWorkflow() *Workflow {
    nodes := make(map[string]*WorkflowNode)
    edges := []*WorkflowEdge{}

    nodes["crawl_old_site"] = &WorkflowNode{
        ID:       "crawl_old_site",
        Type:     "task",
        Name:     "Crawl Old Site",
        TaskType: "crawl",
        Inputs:   []string{"old_site_url"},
        Outputs:  []string{"old_site_urls", "old_site_structure"},
    }

    nodes["url_mapping"] = &WorkflowNode{
        ID:       "url_mapping",
        Type:     "task",
        Name:     "URL Mapping",
        TaskType: "url_mapper",
        Inputs:   []string{"old_site_urls", "new_site_structure"},
        Outputs:  []string{"url_mappings"},
    }

    nodes["redirect_creation"] = &WorkflowNode{
        ID:       "redirect_creation",
        Type:     "task",
        Name:     "Create Redirects",
        TaskType: "redirect_generator",
        Inputs:   []string{"url_mappings"},
        Outputs:  []string{"redirect_rules"},
    }

    nodes["staging_deploy"] = &WorkflowNode{
        ID:       "staging_deploy",
        Type:     "task",
        Name:     "Deploy to Staging",
        TaskType: "deployer",
        Inputs:   []string{"new_site_files"},
        Outputs:  []string{"staging_url"},
    }

    nodes["content_validation"] = &WorkflowNode{
        ID:       "content_validation",
        Type:     "task",
        Name:     "Validate Content",
        TaskType: "content_validator",
        Inputs:   []string{"staging_url", "old_site_urls"},
        Outputs:  []string{"content_matches", "missing_content"},
    }

    nodes["link_checking"] = &WorkflowNode{
        ID:       "link_checking",
        Type:     "task",
        Name:     "Check Links",
        TaskType: "link_analyzer",
        Inputs:   []string{"staging_url"},
        Outputs:  []string{"broken_links", "redirect_chains"},
    }

    nodes["schema_validation"] = &WorkflowNode{
        ID:       "schema_validation",
        Type:     "task",
        Name:     "Validate Schema",
        TaskType: "schema_validator",
        Inputs:   []string{"staging_url"},
        Outputs:  []string{"schema_errors", "schema_warnings"},
    }

    nodes["performance_testing"] = &WorkflowNode{
        ID:       "performance_testing",
        Type:     "task",
        Name:     "Performance Testing",
        TaskType: "performance_tester",
        Inputs:   []string{"staging_url"},
        Outputs:  []string{"performance_metrics", "core_web_vitals"},
    }

    nodes["dns_switch"] = &WorkflowNode{
        ID:       "dns_switch",
        Type:     "human",
        Name:     "DNS Switch Approval",
        TaskType: "approval",
        Inputs:   []string{"validation_results"},
        Outputs:  []string{"dns_approved"},
        Config: map[string]interface{}{
            "assigned_to": "devops",
        },
    }

    nodes["post_migration_crawl"] = &WorkflowNode{
        ID:       "post_migration_crawl",
        Type:     "task",
        Name:     "Post-Migration Crawl",
        TaskType: "crawl",
        Inputs:   []string{"new_site_url"},
        Outputs:  []string{"crawl_results"},
    }

    nodes["error_monitoring"] = &WorkflowNode{
        ID:       "error_monitoring",
        Type:     "task",
        Name:     "Setup Error Monitoring",
        TaskType: "monitoring",
        Inputs:   []string{"new_site_url"},
        Outputs:  []string{"monitoring_setup"},
    }

    // Add start and end nodes
    nodes["start"] = &WorkflowNode{ID: "start", Type: "start", Name: "Start"}
    nodes["end"] = &WorkflowNode{ID: "end", Type: "end", Name: "End"}

    // Define edges
    edges = append(edges,
        &WorkflowEdge{From: "start", To: "crawl_old_site"},
        &WorkflowEdge{From: "crawl_old_site", To: "url_mapping"},
        &WorkflowEdge{From: "url_mapping", To: "redirect_creation"},
        &WorkflowEdge{From: "redirect_creation", To: "staging_deploy"},
        &WorkflowEdge{From: "staging_deploy", To: "content_validation"},
        &WorkflowEdge{From: "content_validation", To: "link_checking"},
        &WorkflowEdge{From: "link_checking", To: "schema_validation"},
        &WorkflowEdge{From: "schema_validation", To: "performance_testing"},
        &WorkflowEdge{From: "performance_testing", To: "dns_switch"},
        &WorkflowEdge{From: "dns_switch", To: "post_migration_crawl"},
        &WorkflowEdge{From: "post_migration_crawl", To: "error_monitoring"},
        &WorkflowEdge{From: "error_monitoring", To: "end"},
    )

    return &Workflow{
        ID:          "site_migration",
        Name:        "Site Migration Workflow",
        Description: "Complete site migration workflow",
        Version:     "1.0.0",
        Nodes:       nodes,
        Edges:       edges,
        CreatedAt:   time.Now(),
        Status:      "active",
        MaxRetries:  3,
        Timeout:     48 * time.Hour,
    }
}