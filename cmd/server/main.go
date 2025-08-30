package main

import (
	"context"
	"log"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/compose"
	"github.com/jontolof/docker-compose-mcp/internal/filter"
	"github.com/jontolof/docker-compose-mcp/internal/mcp"
)

func main() {
	server := mcp.NewServer()
	composeClient := compose.NewClient()
	outputFilter := filter.NewOutputFilter()

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
	}

	return args
}