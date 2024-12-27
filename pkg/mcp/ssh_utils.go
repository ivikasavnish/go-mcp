package mcp

import (
	"bytes"
	"fmt"
	"github.com/pkg/sftp"
	"io"
	_ "io/ioutil"
	"os"
	_ "path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient represents an SSH connection client
type SSHClient struct {
	config    *ssh.ClientConfig
	client    *ssh.Client
	session   *ssh.Session
	host      string
	port      int
	connected bool
	mu        sync.Mutex
}

// SSHConfig represents SSH connection configuration
type SSHConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Password      string `json:"password,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	KeyPassphrase string `json:"key_passphrase,omitempty"`
}

// CommandResult represents the result of an SSH command execution
type CommandResult struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// NewSSHClient creates a new SSH client
func NewSSHClient(config SSHConfig) (*SSHClient, error) {
	var authMethods []ssh.AuthMethod

	// Add password authentication if provided
	if config.Password != "" {
		authMethods = append(authMethods, ssh.Password(config.Password))
	}

	// Add private key authentication if provided
	if config.PrivateKey != "" {
		key, err := parsePrivateKey(config.PrivateKey, config.KeyPassphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method provided")
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	return &SSHClient{
		config: sshConfig,
		host:   config.Host,
		port:   config.Port,
	}, nil
}

// Connect establishes an SSH connection
func (c *SSHClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.config)
	if err != nil {
		return fmt.Errorf("failed to dial: %v", err)
	}

	c.client = client
	c.connected = true
	return nil
}

// Close closes the SSH connection
func (c *SSHClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.session != nil {
		c.session.Close()
		c.session = nil
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %v", err)
	}

	c.connected = false
	return nil
}

// ExecuteCommand executes a command over SSH
func (c *SSHClient) ExecuteCommand(command string) (*CommandResult, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}
	}

	return &CommandResult{
		Command:  command,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// UploadFile uploads a file to the remote server
func (c *SSHClient) UploadFile(localPath, remotePath string) error {
	if err := c.Connect(); err != nil {
		return err
	}

	// Create SFTP client
	sftpClient, err := c.createSFTPClient()
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %v", err)
	}
	defer localFile.Close()

	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %v", err)
	}
	defer remoteFile.Close()

	// Copy file contents
	_, err = io.Copy(remoteFile, localFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	return nil
}

// DownloadFile downloads a file from the remote server
func (c *SSHClient) DownloadFile(remotePath, localPath string) error {
	if err := c.Connect(); err != nil {
		return err
	}

	// Create SFTP client
	sftpClient, err := c.createSFTPClient()
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// Open remote file
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %v", err)
	}
	defer remoteFile.Close()

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %v", err)
	}
	defer localFile.Close()

	// Copy file contents
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	return nil
}

// Helper functions

func parsePrivateKey(keyData string, passphrase string) (ssh.Signer, error) {
	var err error
	var key ssh.Signer

	privateKey := []byte(keyData)

	if passphrase != "" {
		key, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(passphrase))
	} else {
		key, err = ssh.ParsePrivateKey(privateKey)
	}

	if err != nil {
		return nil, err
	}

	return key, nil
}

func (c *SSHClient) createSFTPClient() (*sftp.Client, error) {
	return sftp.NewClient(c.client)
}
