package compose

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Execute(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", &ComposeError{
				Message: strings.TrimSpace(stderr.String()),
				Output:  stdout.String(),
			}
		}
		return "", err
	}
	
	return stdout.String(), nil
}

type ComposeError struct {
	Message string
	Output  string
}

func (e *ComposeError) Error() string {
	return e.Message
}