# MCP Tools Reference

Complete reference for all Docker Compose MCP server tools with intelligent output filtering.

## Core Operations (5 tools)

### compose_up
Start services defined in docker-compose.yml with filtered output.

**Parameters:**
- `services` (array, optional): Specific services to start
- `detach` (boolean, optional): Run in detached mode (default: true)
- `build` (boolean, optional): Build images before starting

**Example:**
```json
{
  "services": ["web", "database"],
  "detach": true,
  "build": false
}
```

**Filtering:** Removes verbose image pulls, build cache operations, preserves service status and errors.

---

### compose_down
Stop and remove containers, networks, and volumes.

**Parameters:**
- `volumes` (boolean, optional): Remove named volumes
- `remove_orphans` (boolean, optional): Remove containers for services not defined in compose file

**Example:**
```json
{
  "volumes": false,
  "remove_orphans": true
}
```

**Filtering:** Shows only service shutdown status and cleanup summary.

---

### compose_ps
List containers with health status and port information.

**Parameters:**
- `all` (boolean, optional): Show all containers including stopped ones

**Example:**
```json
{
  "all": false
}
```

**Returns:** Structured container information with ID, name, status, and ports.

---

### compose_logs
Retrieve logs with intelligent filtering and level-based output.

**Parameters:**
- `services` (array, optional): Specific services to get logs from
- `follow` (boolean, optional): Follow log output
- `tail` (string, optional): Number of lines to show from end

**Example:**
```json
{
  "services": ["web"],
  "follow": false,
  "tail": "100"
}
```

**Filtering:** Preserves errors, warnings, and key application events while removing noise.

---

### compose_build
Build or rebuild services with progress filtering.

**Parameters:**
- `services` (array, optional): Specific services to build
- `no_cache` (boolean, optional): Build without using cache

**Example:**
```json
{
  "services": ["web"],
  "no_cache": false
}
```

**Filtering:** Shows build progress, errors, and warnings while removing verbose layer operations.

## Testing & Development (2 tools)

### compose_exec
Execute commands in running containers with structured output.

**Parameters:**
- `service` (string, **required**): Service name to execute command in
- `command` (string, **required**): Command to execute
- `interactive` (boolean, optional): Enable interactive mode
- `tty` (boolean, optional): Allocate pseudo-TTY
- `user` (string, optional): User to run command as
- `workdir` (string, optional): Working directory

**Example:**
```json
{
  "service": "web",
  "command": "ls -la",
  "interactive": false,
  "tty": false
}
```

---

### compose_test
Run tests in containers with framework-specific output parsing.

**Parameters:**
- `service` (string, **required**): Service to run tests in
- `test_command` (string, optional): Custom test command (default: "go test ./...")
- `test_framework` (string, optional): Framework type (go, jest, pytest)
- `coverage` (boolean, optional): Generate coverage report
- `verbose` (boolean, optional): Verbose test output

**Example:**
```json
{
  "service": "api",
  "test_framework": "go",
  "coverage": true,
  "verbose": false
}
```

**Filtering:** Extracts test results, failures, coverage data, and summary statistics.

## Session-Based Monitoring (3 tools)

### compose_watch_start
Start file watching with automatic rebuild and restart.

**Parameters:**
- `services` (array, optional): Services to watch
- `build` (boolean, optional): Rebuild on changes

**Returns:** Session ID for managing the watch operation.

**Example:**
```json
{
  "services": ["web"],
  "build": true
}
```

---

### compose_watch_stop
Stop an active watch session.

**Parameters:**
- `session_id` (string, **required**): Session ID from watch_start

**Example:**
```json
{
  "session_id": "watch_abc123"
}
```

---

### compose_watch_status
Get status and output from a watch session.

**Parameters:**
- `session_id` (string, **required**): Session ID from watch_start

**Returns:** Session status, start time, and recent output.

## Database Operations (3 tools)

### compose_migrate
Run database migrations with intelligent output filtering.

**Parameters:**
- `service` (string, **required**): Database service name
- `migrate_command` (string, optional): Migration command (default: "migrate up")
- `direction` (string, optional): Migration direction ("up" or "down")
- `steps` (string, optional): Number of migration steps
- `target` (string, optional): Target migration version

**Example:**
```json
{
  "service": "database",
  "direction": "up",
  "steps": "1"
}
```

**Filtering:** Preserves migration status, version changes, and errors while removing connection noise.

---

### compose_db_reset
Reset database to clean state with safety confirmations.

**Parameters:**
- `service` (string, **required**): Database service name
- `confirm` (boolean, **required**): Explicit confirmation required
- `reset_command` (string, optional): Custom reset command
- `seed_command` (string, optional): Seeding command after reset
- `backup_first` (boolean, optional): Create backup before reset

**Example:**
```json
{
  "service": "database",
  "confirm": true,
  "backup_first": true,
  "seed_command": "npm run seed"
}
```

**Safety:** Requires explicit confirmation and optionally creates backup before destructive operations.

---

### compose_db_backup
Create, restore, and list database backups.

**Parameters:**
- `service` (string, **required**): Database service name
- `action` (string, **required**): Action to perform ("create", "restore", "list")
- `backup_name` (string, optional): Backup name (for create/restore)
- `backup_command` (string, optional): Custom backup command
- `restore_command` (string, optional): Custom restore command
- `list_command` (string, optional): Custom list command

**Example:**
```json
{
  "service": "database",
  "action": "create",
  "backup_name": "pre-deployment-backup"
}
```

## Optimization & Performance (1 tool)

### compose_optimization
Get performance metrics, cache statistics, and filtering analytics.

**Parameters:**
- `action` (string, **required**): Action to perform ("stats", "reset", "cache", "export")
- `format` (string, optional): Output format ("json" or "summary")
- `operation` (string, optional): Filter by specific operation

**Example:**
```json
{
  "action": "stats",
  "format": "summary"
}
```

**Actions:**
- **stats**: Get filtering performance metrics
- **reset**: Clear all collected metrics
- **cache**: Get configuration cache statistics
- **export**: Export metrics data in JSON format

**Returns:** Detailed performance analytics including:
- Context reduction ratios (target: 90%+)
- Token savings and cost estimates
- Filter effectiveness by operation
- Cache hit rates and memory usage

## Output Filtering Strategy

All tools implement intelligent filtering to reduce context usage by 90%+ while preserving essential information:

### Always Preserved:
- Error messages and stack traces
- Warning messages
- Service status changes
- Test results and failures
- Migration status and version info
- Container health status

### Always Filtered:
- Verbose Docker image pull operations
- Build cache and layer operations
- Repetitive connection logs
- Debug-level framework output
- Progress bars and status updates
- Timestamp-only log entries

### Framework-Specific Filtering:
- **Go**: Test results, coverage, benchmark data
- **Jest/Node**: Test suites, failures, coverage reports  
- **Python/pytest**: Test outcomes, assertion failures
- **Database**: Migration versions, schema changes

## Usage Patterns

### Development Workflow:
1. `compose_up` - Start services
2. `compose_logs` - Check startup status
3. `compose_exec` - Debug issues
4. `compose_test` - Run test suite
5. `compose_down` - Clean shutdown

### Database Management:
1. `compose_db_backup create` - Safety backup
2. `compose_migrate` - Apply schema changes
3. `compose_db_reset` - Clean state (with confirmation)
4. `compose_migrate up` - Reapply migrations

### Performance Monitoring:
1. `compose_optimization stats` - Check filtering performance
2. `compose_optimization cache` - Review cache efficiency
3. `compose_optimization export` - Generate reports

## Error Handling

All tools implement consistent error handling:
- Structured error responses with context
- Fallback to raw output if filtering fails
- Timeout management for long-running operations
- Graceful degradation on Docker Compose failures