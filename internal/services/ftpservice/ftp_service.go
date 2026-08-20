package ftpservice

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// FTPConfig holds FTP/SFTP connection details
type FTPConfig struct {
	Protocol   string `json:"protocol"` // "ftp" or "sftp"
	Host       string `json:"host"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Port       int    `json:"port"`
	RootPath   string `json:"rootPath"`
	Timeout    int    `json:"timeout"`
}

// FTPClient interface for both FTP and SFTP
type FTPClient interface {
	Connect() error
	Disconnect() error
	ListFiles(path string) ([]string, error)
	ReadFile(path string) (string, error)
	WriteFile(path string, content string) error
	CreateBackup(path string) (string, error)
	RestoreBackup(backupPath string) error
	FileExists(path string) bool
}

// FTPService handles FTP/SFTP operations
type FTPService struct {
	logger     *log.Logger
	config     FTPConfig
	ftpClient  *ftp.ServerConn
	sftpClient *sftp.Client
	sshClient  *ssh.Client
	connected  bool
}

// NewFTPService creates a new FTP service
func NewFTPService(config FTPConfig, logger *log.Logger) *FTPService {
	if config.Port == 0 {
		if config.Protocol == "sftp" {
			config.Port = 22
		} else {
			config.Port = 21
		}
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.RootPath == "" {
		config.RootPath = "/"
	}

	return &FTPService{
		logger: logger,
		config: config,
	}
}

// Connect establishes connection to FTP/SFTP server
func (s *FTPService) Connect() error {
	if s.connected {
		return nil
	}

	if s.config.Protocol == "sftp" {
		return s.connectSFTP()
	}
	return s.connectFTP()
}

// connectSFTP establishes SFTP connection
func (s *FTPService) connectSFTP() error {
	s.logger.Printf("Connecting to SFTP server: %s:%d", s.config.Host, s.config.Port)

	// SSH client config
	sshConfig := &ssh.ClientConfig{
		User: s.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(s.config.Timeout) * time.Second,
	}

	// Connect to SSH
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH: %w", err)
	}
	s.sshClient = conn

	// Create SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	s.sftpClient = client
	s.connected = true

	s.logger.Printf("✅ Connected to SFTP server: %s", s.config.Host)
	return nil
}

// connectFTP establishes FTP connection
func (s *FTPService) connectFTP() error {
	s.logger.Printf("Connecting to FTP server: %s:%d", s.config.Host, s.config.Port)

	// Connect to FTP with timeout
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(time.Duration(s.config.Timeout)*time.Second))
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}

	// Login
	if err := conn.Login(s.config.Username, s.config.Password); err != nil {
		conn.Quit()
		return fmt.Errorf("failed to login to FTP: %w", err)
	}

	s.ftpClient = conn
	s.connected = true

	s.logger.Printf("✅ Connected to FTP server: %s", s.config.Host)
	return nil
}

// Disconnect closes the connection
func (s *FTPService) Disconnect() error {
	if !s.connected {
		return nil
	}

	if s.sftpClient != nil {
		s.sftpClient.Close()
	}
	if s.sshClient != nil {
		s.sshClient.Close()
	}
	if s.ftpClient != nil {
		s.ftpClient.Quit()
	}

	s.connected = false
	s.logger.Println("Disconnected from FTP/SFTP server")
	return nil
}

// ListFiles lists all HTML files in the given path
func (s *FTPService) ListFiles(path string) ([]string, error) {
	if !s.connected {
		if err := s.Connect(); err != nil {
			return nil, err
		}
	}

	var files []string

	if s.sftpClient != nil {
		// SFTP walk
		walker := s.sftpClient.Walk(path)
		for walker.Step() {
			if err := walker.Err(); err != nil {
				continue
			}
			if !walker.Stat().IsDir() {
				name := walker.Stat().Name()
				if s.isHTMLFile(name) {
					files = append(files, walker.Path())
				}
			}
		}
	} else if s.ftpClient != nil {
		// FTP walk
		err := s.walkFTP(path, &files)
		if err != nil {
			return nil, err
		}
	}

	s.logger.Printf("Found %d HTML files in %s", len(files), path)
	return files, nil
}

// walkFTP recursively walks FTP directory
func (s *FTPService) walkFTP(path string, files *[]string) error {
	entries, err := s.ftpClient.List(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name)
		if entry.Type == ftp.EntryTypeFolder {
			if entry.Name != "." && entry.Name != ".." {
				if err := s.walkFTP(fullPath, files); err != nil {
					s.logger.Printf("Warning: Failed to walk %s: %v", fullPath, err)
				}
			}
		} else if s.isHTMLFile(entry.Name) {
			*files = append(*files, fullPath)
		}
	}
	return nil
}

// isHTMLFile checks if file is an HTML file
func (s *FTPService) isHTMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".html" || ext == ".htm" || ext == ".php" || ext == ".phtml"
}

// ReadFile reads a file from server
func (s *FTPService) ReadFile(path string) (string, error) {
	if !s.connected {
		if err := s.Connect(); err != nil {
			return "", err
		}
	}

	var content bytes.Buffer

	if s.sftpClient != nil {
		file, err := s.sftpClient.Open(path)
		if err != nil {
			return "", fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(&content, file)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	} else if s.ftpClient != nil {
		reader, err := s.ftpClient.Retr(path)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve file: %w", err)
		}
		defer reader.Close()

		_, err = io.Copy(&content, reader)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
	}

	return content.String(), nil
}

// WriteFile writes content to a file on server
func (s *FTPService) WriteFile(path string, content string) error {
	if !s.connected {
		if err := s.Connect(); err != nil {
			return err
		}
	}

	reader := bytes.NewReader([]byte(content))

	if s.sftpClient != nil {
		file, err := s.sftpClient.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(file, reader)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	} else if s.ftpClient != nil {
		err := s.ftpClient.Stor(path, reader)
		if err != nil {
			return fmt.Errorf("failed to store file: %w", err)
		}
	}

	s.logger.Printf("✅ Written file: %s", path)
	return nil
}

// CreateBackup creates a backup of a file
func (s *FTPService) CreateBackup(path string) (string, error) {
	content, err := s.ReadFile(path)
	if err != nil {
		return "", err
	}

	backupPath := path + ".seosps.backup." + time.Now().Format("20060102_150405")
	err = s.WriteFile(backupPath, content)
	if err != nil {
		return "", err
	}

	s.logger.Printf("✅ Backup created: %s", backupPath)
	return backupPath, nil
}

// RestoreBackup restores a file from backup
func (s *FTPService) RestoreBackup(backupPath string) error {
	content, err := s.ReadFile(backupPath)
	if err != nil {
		return err
	}

	originalPath := strings.TrimSuffix(backupPath, ".backup.*")
	parts := strings.Split(backupPath, ".seosps.backup.")
	if len(parts) > 0 {
		originalPath = parts[0]
	}

	err = s.WriteFile(originalPath, content)
	if err != nil {
		return err
	}

	s.logger.Printf("✅ Restored from backup: %s", backupPath)
	return nil
}

// FileExists checks if a file exists
func (s *FTPService) FileExists(path string) bool {
	if s.sftpClient != nil {
		_, err := s.sftpClient.Stat(path)
		return err == nil
	}
	if s.ftpClient != nil {
		_, err := s.ftpClient.FileSize(path)
		return err == nil
	}
	return false
}