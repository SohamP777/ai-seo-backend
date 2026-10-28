// pkg/shopify/rollback.go
package shopify

import (
    "context"
    "fmt"
    "log"
    "time"
)

type RollbackManager struct {
    backupMgr BackupManager
    client    *ShopifyClient
    logger    *log.Logger
}

func NewRollbackManager(backupMgr BackupManager, client *ShopifyClient) *RollbackManager {
    return &RollbackManager{
        backupMgr: backupMgr,
        client:    client,
        logger:    log.New(log.Writer(), "[ROLLBACK] ", log.LstdFlags),
    }
}

func (r *RollbackManager) Rollback(ctx context.Context, backupID string) ([]FixResult, error) {
    var results []FixResult
    
    r.logger.Printf("Starting rollback to backup: %s", backupID)
    
    // List backups to verify the backup exists
    backups, err := r.backupMgr.ListBackups()
    if err != nil {
        r.logger.Printf("Error listing backups: %v", err)
        return results, fmt.Errorf("list backups: %w", err)
    }
    
    // Find the specific backup
   // Find the specific backup
backupExists := false
for _, backup := range backups {
    if backupID == backup {
        backupExists = true
        break
    }
}

if !backupExists {
    return results, fmt.Errorf("backup %s not found", backupID)
}

r.logger.Printf("Backup found: %s", backupID)
    
    // Restore assets using the backup manager
    err = r.backupMgr.Restore(backupID)
    if err != nil {
        results = append(results, FixResult{
            Success:   false,
            Action:    "rollback",
            Target:    backupID,
            Error:     err.Error(),
            Message:   "Failed to restore backup",
            Timestamp: time.Now(),
        })
        return results, fmt.Errorf("restore backup: %w", err)
    }
    
    results = append(results, FixResult{
        Success:   true,
        Action:    "rollback",
        Target:    backupID,
        Message:   "Successfully rolled back to previous state",
        Timestamp: time.Now(),
    })
    
    r.logger.Printf("Rollback completed successfully for backup: %s", backupID)
    
    return results, nil
}

func (r *RollbackManager) ListRollbacks(storeID string) ([]*Backup, error) {
    // List all backups
    backups, err := r.backupMgr.ListBackups()
    if err != nil {
        return nil, fmt.Errorf("list backups: %w", err)
    }
    
    // Filter backups by storeID if needed
    var filteredBackups []*Backup
    for _, backup := range backups {
        if backup == storeID || storeID == "" {
            filteredBackups = append(filteredBackups, &Backup{ID: backup})
        }
    }
    return filteredBackups, nil
}  // ← Closing brace for ListRollbacks

// Additional helper method to get backup details
func (r *RollbackManager) GetBackupDetails(backupID string) (*Backup, error) {
    backups, err := r.backupMgr.ListBackups()
    if err != nil {
        return nil, fmt.Errorf("list backups: %w", err)
    }
    
    for _, backup := range backups {
        if backupID == backup {
            // Convert string to *Backup if needed
            return &Backup{
                ID: backup,
                // Add other fields if available
            }, nil
        }
    }
    
    return nil, fmt.Errorf("backup %s not found", backupID)
}  // ← Closing brace for GetBackupDetails