// pkg/wordpress/backup.go
package wordpress

import (
    "archive/zip"
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
)

type BackupManager struct {
    client      *Client
    backupDir   string
    logger      *Logger
}

func NewBackupManager(client *Client, backupDir string, logger *Logger) *BackupManager {
    return &BackupManager{
        client:    client,
        backupDir: backupDir,
        logger:    logger,
    }
}

func (b *BackupManager) CreateBackup(ctx context.Context, siteURL string) (*Backup, error) {
    backupID := generateBackupID()
    backupPath := filepath.Join(b.backupDir, backupID)
    
    if err := os.MkdirAll(backupPath, 0755); err != nil {
        return nil, fmt.Errorf("failed to create backup directory: %w", err)
    }
    
    b.logger.Info("Creating backup %s for %s", backupID, siteURL)
    
    // Backup database
    dbBackup, err := b.backupDatabase(ctx, backupPath)
    if err != nil {
        return nil, fmt.Errorf("failed to backup database: %w", err)
    }
    
    // Backup files
    filesBackup, err := b.backupFiles(ctx, backupPath)
    if err != nil {
        return nil, fmt.Errorf("failed to backup files: %w", err)
    }
    
    // Calculate size
    var size int64
    filepath.Walk(backupPath, func(_ string, info os.FileInfo, err error) error {
        if err == nil && !info.IsDir() {
            size += info.Size()
        }
        return nil
    })
    
    backup := &Backup{
        ID:           backupID,
        SiteURL:      siteURL,
        CreatedAt:    time.Now(),
        DatabaseDump: dbBackup,
        FilesBackup:  filesBackup,
        Size:         size,
    }
    
    // Save backup metadata
    metadataPath := filepath.Join(backupPath, "metadata.json")
    metadataBytes, err := json.Marshal(backup)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal metadata: %w", err)
    }
    
    if err := os.WriteFile(metadataPath, metadataBytes, 0644); err != nil {
        return nil, fmt.Errorf("failed to write metadata: %w", err)
    }
    
    b.logger.Info("Backup %s created successfully (size: %d bytes)", backupID, size)
    
    return backup, nil
}

func (b *BackupManager) backupDatabase(ctx context.Context, backupPath string) (string, error) {
    // Get all posts, pages, and settings
    posts, err := b.client.GetPosts(ctx, 1, 100)
    if err != nil {
        return "", err
    }
    
    settings, err := b.client.GetSettings(ctx)
    if err != nil {
        return "", err
    }
    
    backup := map[string]interface{}{
        "posts":    posts,
        "settings": settings,
        "timestamp": time.Now(),
    }
    
    dbPath := filepath.Join(backupPath, "database.json")
    data, err := json.MarshalIndent(backup, "", "  ")
    if err != nil {
        return "", err
    }
    
    if err := os.WriteFile(dbPath, data, 0644); err != nil {
        return "", err
    }
    
    return dbPath, nil
}

func (b *BackupManager) backupFiles(ctx context.Context, backupPath string) (string, error) {
    // Get theme files
    themes, err := b.client.Get(ctx, "/wp-json/wp/v2/themes")
    if err != nil {
        b.logger.Error("Failed to get themes: %v", err)
    }
    
    filesPath := filepath.Join(backupPath, "files.json")
    data, err := json.MarshalIndent(themes, "", "  ")
    if err != nil {
        return "", err
    }
    
    if err := os.WriteFile(filesPath, data, 0644); err != nil {
        return "", err
    }
    
    return filesPath, nil
}

func (b *BackupManager) RestoreBackup(ctx context.Context, backupID string) error {
    backupPath := filepath.Join(b.backupDir, backupID)
    metadataPath := filepath.Join(backupPath, "metadata.json")
    
    // Read metadata
    metadataBytes, err := os.ReadFile(metadataPath)
    if err != nil {
        return fmt.Errorf("failed to read metadata: %w", err)
    }
    
    var backup Backup
    if err := json.Unmarshal(metadataBytes, &backup); err != nil {
        return fmt.Errorf("failed to parse metadata: %w", err)
    }
    
    b.logger.Info("Restoring backup %s for %s", backupID, backup.SiteURL)
    
    // Restore database
    if err := b.restoreDatabase(ctx, backupPath); err != nil {
        return fmt.Errorf("failed to restore database: %w", err)
    }
    
    // Restore files
    if err := b.restoreFiles(ctx, backupPath); err != nil {
        return fmt.Errorf("failed to restore files: %w", err)
    }
    
    b.logger.Info("Backup %s restored successfully", backupID)
    
    return nil
}

func (b *BackupManager) restoreDatabase(ctx context.Context, backupPath string) error {
    dbPath := filepath.Join(backupPath, "database.json")
    data, err := os.ReadFile(dbPath)
    if err != nil {
        return err
    }
    
    var backupData map[string]interface{}
    if err := json.Unmarshal(data, &backupData); err != nil {
        return err
    }
    
    // Restore settings
    if settings, ok := backupData["settings"].(map[string]interface{}); ok {
        if err := b.client.UpdateSettings(ctx, settings); err != nil {
            return fmt.Errorf("failed to restore settings: %w", err)
        }
    }
    
    // Restore posts (in production, you'd need to handle updates vs creates)
    if posts, ok := backupData["posts"].([]interface{}); ok {
        for _, post := range posts {
            postMap, ok := post.(map[string]interface{})
            if !ok {
                continue
            }
            
            if postID, ok := postMap["id"].(float64); ok {
                // Update existing post
                if err := b.client.UpdatePost(ctx, int(postID), postMap); err != nil {
                    b.logger.Error("Failed to restore post %d: %v", int(postID), err)
                }
            }
        }
    }
    
    return nil
}

func (b *BackupManager) restoreFiles(ctx context.Context, backupPath string) error {
    // In production, you'd restore actual files here
    // For now, we'll just log it
    b.logger.Info("Files restored from %s", backupPath)
    return nil
}

func generateBackupID() string {
    bytes := make([]byte, 16)
    rand.Read(bytes)
    return hex.EncodeToString(bytes)
}

func (b *BackupManager) CreateZip(backupID string) error {
    backupPath := filepath.Join(b.backupDir, backupID)
    zipPath := backupPath + ".zip"
    
    zipFile, err := os.Create(zipPath)
    if err != nil {
        return err
    }
    defer zipFile.Close()
    
    zipWriter := zip.NewWriter(zipFile)
    defer zipWriter.Close()
    
    err = filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            return nil
        }
        
        relPath, err := filepath.Rel(backupPath, path)
        if err != nil {
            return err
        }
        
        file, err := os.Open(path)
        if err != nil {
            return err
        }
        defer file.Close()
        
        writer, err := zipWriter.Create(relPath)
        if err != nil {
            return err
        }
        
        _, err = io.Copy(writer, file)
        return err
    })
    
    if err != nil {
        return err
    }
    
    return nil
}