package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/compose"
	"github.com/jontolof/docker-compose-mcp/internal/filter"
	"github.com/jontolof/docker-compose-mcp/internal/mcp"
	"github.com/jontolof/docker-compose-mcp/internal/session"
)

func main() {
	server := mcp.NewServer()
	composeClient := compose.NewClient()
	outputFilter := filter.NewOutputFilter()
	sessionManager := session.NewManager()

	server.RegisterTool(mcp.Tool{
		Name:        "compose_up",
		Description: "Start services defined in docker-compose.yml",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"services": {
					Type: "array",
				},
				"detach": {
					Type: "boolean",
				},
				"build": {
					Type: "boolean",
				},
			},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleComposeCommand(composeClient, outputFilter, "up", params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_down",
		Description: "Stop and remove containers, networks, and volumes",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"volumes": {
					Type: "boolean",
				},
				"remove_orphans": {
					Type: "boolean",
				},
			},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleComposeCommand(composeClient, outputFilter, "down", params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_ps",
		Description: "List containers",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"all": {
					Type: "boolean",
				},
			},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleComposeCommand(composeClient, outputFilter, "ps", params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_logs",
		Description: "View output from containers",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"services": {
					Type: "array",
				},
				"follow": {
					Type: "boolean",
				},
				"tail": {
					Type: "string",
				},
			},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleComposeCommand(composeClient, outputFilter, "logs", params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_build",
		Description: "Build or rebuild services",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"services": {
					Type: "array",
				},
				"no_cache": {
					Type: "boolean",
				},
			},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleComposeCommand(composeClient, outputFilter, "build", params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_exec",
		Description: "Execute a command in a running container",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {
					Type: "string",
				},
				"command": {
					Type: "string",
				},
				"interactive": {
					Type: "boolean",
				},
				"tty": {
					Type: "boolean",
				},
				"user": {
					Type: "string",
				},
				"workdir": {
					Type: "string",
				},
			},
			Required: []string{"service", "command"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleComposeCommand(composeClient, outputFilter, "exec", params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_test",
		Description: "Run tests in containers with intelligent output parsing",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {
					Type: "string",
				},
				"test_command": {
					Type: "string",
				},
				"test_framework": {
					Type: "string",
				},
				"coverage": {
					Type: "boolean",
				},
				"verbose": {
					Type: "boolean",
				},
			},
			Required: []string{"service"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleTestCommand(composeClient, outputFilter, params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_watch_start",
		Description: "Start watching containers for changes and rebuild/restart automatically",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"services": {
					Type: "array",
				},
				"build": {
					Type: "boolean",
				},
			},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleWatchStart(sessionManager, composeClient, outputFilter, params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_watch_stop",
		Description: "Stop a running watch session",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"session_id": {
					Type: "string",
				},
			},
			Required: []string{"session_id"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleWatchStop(sessionManager, params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_watch_status",
		Description: "Get status and output from a watch session",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"session_id": {
					Type: "string",
				},
			},
			Required: []string{"session_id"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleWatchStatus(sessionManager, params)
		},
	})

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}

func handleComposeCommand(client *compose.Client, filter *filter.OutputFilter, command string, params interface{}) (interface{}, error) {
	args := []string{command}

	if params != nil {
		if paramsMap, ok := params.(map[string]interface{}); ok {
			args = append(args, buildArgs(command, paramsMap)...)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	output, err := client.Execute(ctx, args)
	if err != nil {
		if composeErr, ok := err.(*compose.ComposeError); ok {
			filtered := filter.Filter(composeErr.Output + "\n" + composeErr.Message)
			return filtered, nil
		}
		return err.Error(), nil
	}

	filtered := filter.Filter(output)
	return filtered, nil
}

func buildArgs(command string, params map[string]interface{}) []string {
	var args []string

	switch command {
	case "up":
		if detach, ok := params["detach"].(bool); ok && detach {
			args = append(args, "-d")
		}
		if build, ok := params["build"].(bool); ok && build {
			args = append(args, "--build")
		}
		if services, ok := params["services"].([]interface{}); ok {
			for _, service := range services {
				if serviceName, ok := service.(string); ok {
					args = append(args, serviceName)
				}
			}
		}

	case "down":
		if volumes, ok := params["volumes"].(bool); ok && volumes {
			args = append(args, "-v")
		}
		if removeOrphans, ok := params["remove_orphans"].(bool); ok && removeOrphans {
			args = append(args, "--remove-orphans")
		}

	case "ps":
		if all, ok := params["all"].(bool); ok && all {
			args = append(args, "-a")
		}

	case "logs":
		if follow, ok := params["follow"].(bool); ok && follow {
			args = append(args, "-f")
		}
		if tail, ok := params["tail"].(string); ok && tail != "" {
			args = append(args, "--tail", tail)
		}
		if services, ok := params["services"].([]interface{}); ok {
			for _, service := range services {
				if serviceName, ok := service.(string); ok {
					args = append(args, serviceName)
				}
			}
		}

	case "build":
		if noCache, ok := params["no_cache"].(bool); ok && noCache {
			args = append(args, "--no-cache")
		}
		if services, ok := params["services"].([]interface{}); ok {
			for _, service := range services {
				if serviceName, ok := service.(string); ok {
					args = append(args, serviceName)
				}
			}
		}

	case "exec":
		if interactive, ok := params["interactive"].(bool); ok && interactive {
			args = append(args, "-i")
		}
		if tty, ok := params["tty"].(bool); ok && tty {
			args = append(args, "-t")
		}
		if user, ok := params["user"].(string); ok && user != "" {
			args = append(args, "--user", user)
		}
		if workdir, ok := params["workdir"].(string); ok && workdir != "" {
			args = append(args, "--workdir", workdir)
		}
		
		if service, ok := params["service"].(string); ok && service != "" {
			args = append(args, service)
		}
		
		if command, ok := params["command"].(string); ok && command != "" {
			args = append(args, strings.Split(command, " ")...)
		}
	}

	return args
}

func handleTestCommand(client *compose.Client, filter *filter.OutputFilter, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "Invalid parameters", nil
	}

	service, ok := paramsMap["service"].(string)
	if !ok || service == "" {
		return "Service name is required", nil
	}

	testCommand := "go test ./..."
	if cmd, ok := paramsMap["test_command"].(string); ok && cmd != "" {
		testCommand = cmd
	}

	framework := "go"
	if fw, ok := paramsMap["test_framework"].(string); ok && fw != "" {
		framework = fw
	}

	args := []string{"exec", "-T", service}

	if coverage, ok := paramsMap["coverage"].(bool); ok && coverage {
		switch framework {
		case "go":
			testCommand = "go test -coverprofile=coverage.out ./..."
		case "jest", "node":
			testCommand = "npm test -- --coverage"
		case "pytest", "python":
			testCommand = "pytest --cov=."
		}
	}

	if verbose, ok := paramsMap["verbose"].(bool); ok && verbose {
		switch framework {
		case "go":
			if !strings.Contains(testCommand, "-v") {
				testCommand = strings.Replace(testCommand, "go test", "go test -v", 1)
			}
		case "jest", "node":
			if !strings.Contains(testCommand, "--verbose") {
				testCommand += " --verbose"
			}
		case "pytest", "python":
			if !strings.Contains(testCommand, "-v") {
				testCommand += " -v"
			}
		}
	}

	args = append(args, strings.Split(testCommand, " ")...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	output, err := client.Execute(ctx, args)
	if err != nil {
		if composeErr, ok := err.(*compose.ComposeError); ok {
			filtered := filter.FilterTestOutput(composeErr.Output+"\n"+composeErr.Message, framework)
			return filtered, nil
		}
		return err.Error(), nil
	}

	filtered := filter.FilterTestOutput(output, framework)
	return filtered, nil
}

func handleWatchStart(sessionMgr *session.Manager, client *compose.Client, filter *filter.OutputFilter, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		paramsMap = make(map[string]interface{})
	}

	sess := sessionMgr.CreateSession("watch")
	
	go func() {
		defer close(sess.Output)
		
		args := []string{"watch"}
		
		if build, ok := paramsMap["build"].(bool); ok && build {
			args = append(args, "--build")
		}
		
		if services, ok := paramsMap["services"].([]interface{}); ok {
			for _, service := range services {
				if serviceName, ok := service.(string); ok {
					args = append(args, serviceName)
				}
			}
		}

		output, err := client.Execute(sess.Context, args)
		if err != nil {
			sess.Output <- "Watch failed: " + err.Error()
			return
		}
		
		filtered := filter.Filter(output)
		sess.Output <- filtered
	}()

	return map[string]interface{}{
		"session_id": sess.ID,
		"status":     "started",
		"message":    "Watch session started successfully",
	}, nil
}

func handleWatchStop(sessionMgr *session.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "Invalid parameters", nil
	}

	sessionID, ok := paramsMap["session_id"].(string)
	if !ok || sessionID == "" {
		return "Session ID is required", nil
	}

	err := sessionMgr.StopSession(sessionID)
	if err != nil {
		return "Failed to stop session: " + err.Error(), nil
	}

	return map[string]interface{}{
		"session_id": sessionID,
		"status":     "stopped",
		"message":    "Watch session stopped successfully",
	}, nil
}

func handleWatchStatus(sessionMgr *session.Manager, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "Invalid parameters", nil
	}

	sessionID, ok := paramsMap["session_id"].(string)
	if !ok || sessionID == "" {
		return "Session ID is required", nil
	}

	sess, exists := sessionMgr.GetSession(sessionID)
	if !exists {
		return "Session not found", nil
	}

	var output []string
	select {
	case msg := <-sess.Output:
		output = append(output, msg)
	default:
	}

	return map[string]interface{}{
		"session_id": sess.ID,
		"status":     "running",
		"start_time": sess.StartTime.Format(time.RFC3339),
		"output":     output,
	}, nil
}