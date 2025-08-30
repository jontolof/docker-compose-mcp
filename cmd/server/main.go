package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/jontolof/docker-compose-mcp/internal/compose"
	"github.com/jontolof/docker-compose-mcp/internal/config"
	"github.com/jontolof/docker-compose-mcp/internal/errors"
	"github.com/jontolof/docker-compose-mcp/internal/filter"
	"github.com/jontolof/docker-compose-mcp/internal/logging"
	"github.com/jontolof/docker-compose-mcp/internal/mcp"
	"github.com/jontolof/docker-compose-mcp/internal/session"
	"github.com/jontolof/docker-compose-mcp/internal/shutdown"
	"github.com/jontolof/docker-compose-mcp/internal/tools"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger := logging.NewLogger("main", logging.ErrorLevel, false)
		logger.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Initialize logging
	logger := logging.NewLogger("main", cfg.LogLevel, cfg.LogFormat == "json")
	if cfg.LogFile != "" {
		fileLogger, err := logging.NewFileLogger("main", cfg.LogLevel, cfg.LogFormat == "json", cfg.LogFile)
		if err != nil {
			logger.Warnf("Failed to create file logger: %v", err)
		} else {
			logger = fileLogger
		}
	}

	// Initialize error handler
	errorHandler := errors.NewErrorHandler(errors.LogLevel(cfg.LogLevel))

	// Initialize shutdown manager
	shutdownMgr := shutdown.NewManager(cfg.ShutdownTimeout)
	shutdownMgr.Listen()

	logger.Infof("Starting Docker Compose MCP Server")
	logger.Infof("Configuration loaded - Cache: %v, Metrics: %v, Parallel: %v", 
		cfg.EnableCache, cfg.EnableMetrics, cfg.EnableParallel)

	// Initialize components
	server := mcp.NewServer()
	composeClient := compose.NewClient(&compose.ClientOptions{
		WorkDir:         cfg.WorkDir,
		EnableCache:     cfg.EnableCache,
		EnableMetrics:   cfg.EnableMetrics,
		EnableParallel:  cfg.EnableParallel,
		CacheSize:       cfg.CacheSize,
		CacheMaxAge:     cfg.CacheMaxAge,
		MaxWorkers:      cfg.MaxWorkers,
		CommandTimeout:  cfg.CommandTimeout,
	})
	outputFilter := filter.NewOutputFilter()
	sessionManager := session.NewManager()
	optimizationTool := tools.NewOptimizationTool(composeClient)

	// Register cleanup handlers
	shutdownMgr.RegisterSessionCleanup(sessionManager)
	shutdownMgr.RegisterResource("compose_client", composeClient)
	shutdownMgr.AddHandlerFunc("log_shutdown", func() error {
		logger.Info("Shutting down Docker Compose MCP Server")
		return nil
	})

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

	server.RegisterTool(mcp.Tool{
		Name:        "compose_migrate",
		Description: "Run database migrations in a service container",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {
					Type: "string",
				},
				"migrate_command": {
					Type: "string",
				},
				"direction": {
					Type: "string",
				},
				"steps": {
					Type: "string",
				},
				"target": {
					Type: "string",
				},
			},
			Required: []string{"service"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleMigrateCommand(composeClient, outputFilter, params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_db_reset",
		Description: "Reset database to clean state, optionally with seeds",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {
					Type: "string",
				},
				"reset_command": {
					Type: "string",
				},
				"seed_command": {
					Type: "string",
				},
				"confirm": {
					Type: "boolean",
				},
				"backup_first": {
					Type: "boolean",
				},
			},
			Required: []string{"service", "confirm"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleDbResetCommand(composeClient, outputFilter, params)
		},
	})

	server.RegisterTool(mcp.Tool{
		Name:        "compose_db_backup",
		Description: "Create, restore, or list database backups",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {
					Type: "string",
				},
				"action": {
					Type: "string",
				},
				"backup_name": {
					Type: "string",
				},
				"backup_command": {
					Type: "string",
				},
				"restore_command": {
					Type: "string",
				},
				"list_command": {
					Type: "string",
				},
			},
			Required: []string{"service", "action"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return handleDbBackupCommand(composeClient, outputFilter, params)
		},
	})

	// Register optimization tool
	server.RegisterTool(mcp.Tool{
		Name:        "compose_optimization",
		Description: optimizationTool.GetDescription(),
		InputSchema: mcp.Schema{
			Type:       "object",
			Properties: convertSchemaProperties(optimizationTool.GetSchema()["properties"].(map[string]interface{})),
			Required:   []string{"action"},
		},
		Handler: func(params interface{}) (interface{}, error) {
			return optimizationTool.Execute(context.Background(), params)
		},
	})

	// Run server with graceful shutdown
	go func() {
		if err := server.Run(); err != nil {
			errorHandler.Handle(err)
			shutdownMgr.Shutdown()
		}
	}()

	logger.Info("Docker Compose MCP Server started successfully")
	
	// Wait for shutdown
	<-shutdownMgr.Done()
	logger.Info("Server shutdown complete")
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

func handleMigrateCommand(client *compose.Client, filter *filter.OutputFilter, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "Invalid parameters", nil
	}

	service, ok := paramsMap["service"].(string)
	if !ok || service == "" {
		return "Service name is required", nil
	}

	migrateCommand := "migrate up"
	if cmd, ok := paramsMap["migrate_command"].(string); ok && cmd != "" {
		migrateCommand = cmd
	}

	if direction, ok := paramsMap["direction"].(string); ok && direction != "" {
		switch direction {
		case "up", "down":
			if strings.Contains(migrateCommand, "migrate") {
				migrateCommand = strings.Replace(migrateCommand, "migrate", "migrate "+direction, 1)
			} else {
				migrateCommand = "migrate " + direction + " " + migrateCommand
			}
		}
	}

	if steps, ok := paramsMap["steps"].(string); ok && steps != "" {
		migrateCommand += " " + steps
	}

	if target, ok := paramsMap["target"].(string); ok && target != "" {
		migrateCommand += " " + target
	}

	args := []string{"exec", "-T", service}
	args = append(args, strings.Split(migrateCommand, " ")...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	output, err := client.Execute(ctx, args)
	if err != nil {
		if composeErr, ok := err.(*compose.ComposeError); ok {
			filtered := filter.FilterMigrationOutput(composeErr.Output + "\n" + composeErr.Message)
			return filtered, nil
		}
		return err.Error(), nil
	}

	filtered := filter.FilterMigrationOutput(output)
	return filtered, nil
}

func handleDbResetCommand(client *compose.Client, filter *filter.OutputFilter, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "Invalid parameters", nil
	}

	service, ok := paramsMap["service"].(string)
	if !ok || service == "" {
		return "Service name is required", nil
	}

	confirm, ok := paramsMap["confirm"].(bool)
	if !ok || !confirm {
		return "Database reset requires explicit confirmation. Set 'confirm' to true.", nil
	}

	var results []string

	if backupFirst, ok := paramsMap["backup_first"].(bool); ok && backupFirst {
		backupParams := map[string]interface{}{
			"service": service,
			"action":  "create",
			"backup_name": "pre-reset-" + time.Now().Format("20060102-150405"),
		}
		backupResult, err := handleDbBackupCommand(client, filter, backupParams)
		if err != nil {
			return "Failed to create backup before reset: " + err.Error(), nil
		}
		results = append(results, "Backup created: "+backupResult.(string))
	}

	resetCommand := "migrate drop && migrate up"
	if cmd, ok := paramsMap["reset_command"].(string); ok && cmd != "" {
		resetCommand = cmd
	}

	args := []string{"exec", "-T", service}
	args = append(args, strings.Split(resetCommand, " ")...)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	output, err := client.Execute(ctx, args)
	if err != nil {
		if composeErr, ok := err.(*compose.ComposeError); ok {
			filtered := filter.FilterMigrationOutput(composeErr.Output + "\n" + composeErr.Message)
			results = append(results, "Reset failed: "+filtered)
			return strings.Join(results, "\n"), nil
		}
		results = append(results, "Reset failed: "+err.Error())
		return strings.Join(results, "\n"), nil
	}

	filtered := filter.FilterMigrationOutput(output)
	results = append(results, "Database reset completed: "+filtered)

	if seedCommand, ok := paramsMap["seed_command"].(string); ok && seedCommand != "" {
		seedArgs := []string{"exec", "-T", service}
		seedArgs = append(seedArgs, strings.Split(seedCommand, " ")...)

		seedOutput, err := client.Execute(ctx, seedArgs)
		if err != nil {
			results = append(results, "Seeding failed: "+err.Error())
		} else {
			seedFiltered := filter.Filter(seedOutput)
			results = append(results, "Database seeded: "+seedFiltered)
		}
	}

	return strings.Join(results, "\n"), nil
}

func handleDbBackupCommand(client *compose.Client, filter *filter.OutputFilter, params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return "Invalid parameters", nil
	}

	service, ok := paramsMap["service"].(string)
	if !ok || service == "" {
		return "Service name is required", nil
	}

	action, ok := paramsMap["action"].(string)
	if !ok || action == "" {
		return "Action is required (create, restore, list)", nil
	}

	var command string
	var args []string

	switch action {
	case "create":
		backupName := "backup-" + time.Now().Format("20060102-150405")
		if name, ok := paramsMap["backup_name"].(string); ok && name != "" {
			backupName = name
		}

		command = "pg_dump -U $POSTGRES_USER $POSTGRES_DB > /backups/" + backupName + ".sql"
		if cmd, ok := paramsMap["backup_command"].(string); ok && cmd != "" {
			command = strings.Replace(cmd, "{backup_name}", backupName, -1)
		}

		args = []string{"exec", "-T", service, "sh", "-c", command}

	case "restore":
		backupName, ok := paramsMap["backup_name"].(string)
		if !ok || backupName == "" {
			return "Backup name is required for restore action", nil
		}

		command = "psql -U $POSTGRES_USER -d $POSTGRES_DB < /backups/" + backupName + ".sql"
		if cmd, ok := paramsMap["restore_command"].(string); ok && cmd != "" {
			command = strings.Replace(cmd, "{backup_name}", backupName, -1)
		}

		args = []string{"exec", "-T", service, "sh", "-c", command}

	case "list":
		command = "ls -la /backups/"
		if cmd, ok := paramsMap["list_command"].(string); ok && cmd != "" {
			command = cmd
		}

		args = []string{"exec", "-T", service, "sh", "-c", command}

	default:
		return "Invalid action. Use 'create', 'restore', or 'list'", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	output, err := client.Execute(ctx, args)
	if err != nil {
		if composeErr, ok := err.(*compose.ComposeError); ok {
			filtered := filter.Filter(composeErr.Output + "\n" + composeErr.Message)
			return "Backup operation failed: " + filtered, nil
		}
		return "Backup operation failed: " + err.Error(), nil
	}

	filtered := filter.Filter(output)
	return action + " completed successfully: " + filtered, nil
}

func convertSchemaProperties(props map[string]interface{}) map[string]mcp.Schema {
	result := make(map[string]mcp.Schema)
	for key, value := range props {
		if propMap, ok := value.(map[string]interface{}); ok {
			schema := mcp.Schema{}
			if propType, exists := propMap["type"].(string); exists {
				schema.Type = propType
			}
			result[key] = schema
		}
	}
	return result
}