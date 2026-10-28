package utils

import (
    "bufio"
    "crypto/md5"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// FileInfo represents file information
type FileInfo struct {
    Name      string    `json:"name"`
    Path      string    `json:"path"`
    Size      int64     `json:"size"`
    ModTime   time.Time `json:"mod_time"`
    IsDir     bool      `json:"is_dir"`
    Extension string    `json:"extension"`
    MD5       string    `json:"md5,omitempty"`
    SHA256    string    `json:"sha256,omitempty"`
}

// FileUtils provides file operations
type FileUtils struct {
    BasePath string
    MaxSize  int64
    AllowedExts map[string]bool
}

// NewFileUtils creates a new file utils instance
func NewFileUtils(basePath string) *FileUtils {
    return &FileUtils{
        BasePath: basePath,
        MaxSize:  10 * 1024 * 1024, // 10MB default
        AllowedExts: map[string]bool{
            ".txt":  true,
            ".csv":  true,
            ".json": true,
            ".xml":  true,
            ".html": true,
            ".htm":  true,
            ".md":   true,
        },
    }
}

// SaveUploadedFile saves a multipart file to disk
func (fu *FileUtils) SaveUploadedFile(file multipart.File, header *multipart.FileHeader, subDir string) (*FileInfo, error) {
    // Check file size
    if header.Size > fu.MaxSize {
        return nil, fmt.Errorf("file too large: %d bytes (max: %d)", header.Size, fu.MaxSize)
    }
    
    // Check extension
    ext := strings.ToLower(filepath.Ext(header.Filename))
    if !fu.AllowedExts[ext] {
        return nil, fmt.Errorf("file type not allowed: %s", ext)
    }
    
    // Create directory if not exists
    uploadDir := filepath.Join(fu.BasePath, subDir)
    if err := os.MkdirAll(uploadDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create directory: %w", err)
    }
    
    // Generate unique filename
    timestamp := time.Now().UnixNano()
    safeName := fmt.Sprintf("%d_%s", timestamp, sanitizeFilename(header.Filename))
    filePath := filepath.Join(uploadDir, safeName)
    
    // Create destination file
    dst, err := os.Create(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to create file: %w", err)
    }
    defer dst.Close()
    
    // Copy content
    if _, err := io.Copy(dst, file); err != nil {
        os.Remove(filePath) // Clean up on error
        return nil, fmt.Errorf("failed to save file: %w", err)
    }
    
    // Get file info
    return fu.GetFileInfo(filePath)
}

// ReadFile reads entire file content
func (fu *FileUtils) ReadFile(filePath string) ([]byte, error) {
    // Security check - prevent directory traversal
    if !fu.isPathSafe(filePath) {
        return nil, fmt.Errorf("invalid file path")
    }
    
    return os.ReadFile(filePath)
}

// ReadFileLines reads file line by line
func (fu *FileUtils) ReadFileLines(filePath string) ([]string, error) {
    if !fu.isPathSafe(filePath) {
        return nil, fmt.Errorf("invalid file path")
    }
    
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    
    return lines, scanner.Err()
}

// WriteFile writes data to file
func (fu *FileUtils) WriteFile(filePath string, data []byte) error {
    if !fu.isPathSafe(filePath) {
        return fmt.Errorf("invalid file path")
    }
    
    // Create directory if needed
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    return os.WriteFile(filePath, data, 0644)
}

// AppendToFile appends data to file
func (fu *FileUtils) AppendToFile(filePath string, data []byte) error {
    if !fu.isPathSafe(filePath) {
        return fmt.Errorf("invalid file path")
    }
    
    file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = file.Write(data)
    return err
}

// DeleteFile deletes a file
func (fu *FileUtils) DeleteFile(filePath string) error {
    if !fu.isPathSafe(filePath) {
        return fmt.Errorf("invalid file path")
    }
    
    return os.Remove(filePath)
}

// ListFiles lists files in directory
func (fu *FileUtils) ListFiles(dir string) ([]FileInfo, error) {
    fullPath := filepath.Join(fu.BasePath, dir)
    
    if !fu.isPathSafe(fullPath) {
        return nil, fmt.Errorf("invalid directory path")
    }
    
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        return nil, err
    }
    
    var files []FileInfo
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        fileInfo := FileInfo{
            Name:      info.Name(),
            Path:      filepath.Join(dir, info.Name()),
            Size:      info.Size(),
            ModTime:   info.ModTime(),
            IsDir:     info.IsDir(),
            Extension: filepath.Ext(info.Name()),
        }
        
        files = append(files, fileInfo)
    }
    
    return files, nil
}

// GetFileInfo returns file information
func (fu *FileUtils) GetFileInfo(filePath string) (*FileInfo, error) {
    if !fu.isPathSafe(filePath) {
        return nil, fmt.Errorf("invalid file path")
    }
    
    info, err := os.Stat(filePath)
    if err != nil {
        return nil, err
    }
    
    fileInfo := &FileInfo{
        Name:      info.Name(),
        Path:      filePath,
        Size:      info.Size(),
        ModTime:   info.ModTime(),
        IsDir:     info.IsDir(),
        Extension: filepath.Ext(info.Name()),
    }
    
    // Calculate hashes for files only
    if !info.IsDir() {
        if md5, err := fu.CalculateMD5(filePath); err == nil {
            fileInfo.MD5 = md5
        }
        if sha256, err := fu.CalculateSHA256(filePath); err == nil {
            fileInfo.SHA256 = sha256
        }
    }
    
    return fileInfo, nil
}

// CalculateMD5 calculates MD5 hash of file
func (fu *FileUtils) CalculateMD5(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    hash := md5.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }
    
    return hex.EncodeToString(hash.Sum(nil)), nil
}

// CalculateSHA256 calculates SHA256 hash of file
func (fu *FileUtils) CalculateSHA256(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }
    
    return hex.EncodeToString(hash.Sum(nil)), nil
}

// CopyFile copies a file
func (fu *FileUtils) CopyFile(src, dst string) error {
    if !fu.isPathSafe(src) || !fu.isPathSafe(dst) {
        return fmt.Errorf("invalid file path")
    }
    
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()
    
    // Create destination directory
    if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
        return err
    }
    
    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer destFile.Close()
    
    _, err = io.Copy(destFile, sourceFile)
    return err
}

// MoveFile moves/renames a file
func (fu *FileUtils) MoveFile(src, dst string) error {
    if !fu.isPathSafe(src) || !fu.isPathSafe(dst) {
        return fmt.Errorf("invalid file path")
    }
    
    // Create destination directory
    if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
        return err
    }
    
    return os.Rename(src, dst)
}

// FileExists checks if file exists
func (fu *FileUtils) FileExists(filePath string) bool {
    if !fu.isPathSafe(filePath) {
        return false
    }
    
    info, err := os.Stat(filePath)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

// DirExists checks if directory exists
func (fu *FileUtils) DirExists(dirPath string) bool {
    if !fu.isPathSafe(dirPath) {
        return false
    }
    
    info, err := os.Stat(dirPath)
    if os.IsNotExist(err) {
        return false
    }
    return info.IsDir()
}

// CreateTempFile creates a temporary file
func (fu *FileUtils) CreateTempFile(content []byte, pattern string) (string, error) {
    tmpFile, err := os.CreateTemp("", pattern)
    if err != nil {
        return "", err
    }
    defer tmpFile.Close()
    
    if _, err := tmpFile.Write(content); err != nil {
        os.Remove(tmpFile.Name())
        return "", err
    }
    
    return tmpFile.Name(), nil
}

// ReadJSON reads and parses JSON file
func (fu *FileUtils) ReadJSON(filePath string, v interface{}) error {
    data, err := fu.ReadFile(filePath)
    if err != nil {
        return err
    }
    
    return json.Unmarshal(data, v)
}

// WriteJSON writes JSON to file
func (fu *FileUtils) WriteJSON(filePath string, v interface{}, pretty bool) error {
    var data []byte
    var err error
    
    if pretty {
        data, err = json.MarshalIndent(v, "", "  ")
    } else {
        data, err = json.Marshal(v)
    }
    
    if err != nil {
        return err
    }
    
    return fu.WriteFile(filePath, data)
}

// GetFileSize returns human-readable file size
func (fu *FileUtils) GetFileSize(size int64) string {
    const unit = 1024
    if size < unit {
        return fmt.Sprintf("%d B", size)
    }
    
    div, exp := int64(unit), 0
    for n := size / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    
    return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// CleanupOldFiles removes files older than duration
func (fu *FileUtils) CleanupOldFiles(dir string, age time.Duration) (int, error) {
    fullPath := filepath.Join(fu.BasePath, dir)
    
    if !fu.isPathSafe(fullPath) {
        return 0, fmt.Errorf("invalid directory path")
    }
    
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        return 0, err
    }
    
    cutoff := time.Now().Add(-age)
    removed := 0
    
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        if info.ModTime().Before(cutoff) {
            filePath := filepath.Join(fullPath, info.Name())
            if err := os.Remove(filePath); err == nil {
                removed++
            }
        }
    }
    
    return removed, nil
}

// isPathSafe checks if path is within base directory (prevents directory traversal)
func (fu *FileUtils) isPathSafe(requestPath string) bool {
    cleanPath := filepath.Clean(requestPath)
    
    // If path is absolute, ensure it's within BasePath
    if filepath.IsAbs(cleanPath) {
        return strings.HasPrefix(cleanPath, filepath.Clean(fu.BasePath))
    }
    
    // For relative paths, join with BasePath and check
    fullPath := filepath.Join(fu.BasePath, cleanPath)
    return strings.HasPrefix(fullPath, filepath.Clean(fu.BasePath))
}

// sanitizeFilename removes dangerous characters from filename
func sanitizeFilename(filename string) string {
    // Remove path separators
    filename = filepath.Base(filename)
    
    // Replace dangerous characters
    dangerous := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", ".."}
    for _, char := range dangerous {
        filename = strings.ReplaceAll(filename, char, "_")
    }
    
    // Trim spaces
    filename = strings.TrimSpace(filename)
    
    if filename == "" || filename == "." || filename == ".." {
        return "unnamed_file"
    }
    
    return filename
}

// BatchFileProcessor processes multiple files
type BatchFileProcessor struct {
    FileUtils *FileUtils
    OnProgress func(processed, total int)
}

// ProcessFiles processes multiple files with a callback
func (bfp *BatchFileProcessor) ProcessFiles(filePaths []string, processor func(string) error) error {
    total := len(filePaths)
    
    for i, path := range filePaths {
        if err := processor(path); err != nil {
            return fmt.Errorf("error processing %s: %w", path, err)
        }
        
        if bfp.OnProgress != nil {
            bfp.OnProgress(i+1, total)
        }
    }
    
    return nil
}

// FileWatcher watches for file changes
type FileWatcher struct {
    FilePath string
    Callback func()
    ticker   *time.Ticker
    done     chan bool
    lastMod  time.Time
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(filePath string, interval time.Duration, callback func()) *FileWatcher {
    return &FileWatcher{
        FilePath: filePath,
        Callback: callback,
        ticker:   time.NewTicker(interval),
        done:     make(chan bool),
    }
}

// Start starts watching for file changes
func (fw *FileWatcher) Start() error {
    info, err := os.Stat(fw.FilePath)
    if err != nil {
        return err
    }
    fw.lastMod = info.ModTime()
    
    go func() {
        for {
            select {
            case <-fw.ticker.C:
                fw.check()
            case <-fw.done:
                return
            }
        }
    }()
    
    return nil
}

// Stop stops watching
func (fw *FileWatcher) Stop() {
    fw.ticker.Stop()
    fw.done <- true
}

func (fw *FileWatcher) check() {
    info, err := os.Stat(fw.FilePath)
    if err != nil {
        return
    }
    
    if info.ModTime().After(fw.lastMod) {
        fw.lastMod = info.ModTime()
        if fw.Callback != nil {
            fw.Callback()
        }
    }
}