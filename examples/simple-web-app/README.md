# Simple Web App - Docker Compose MCP Demo

This is a complete multi-service web application designed to demonstrate the capabilities of the Docker Compose MCP Server.

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   nginx     │    │   Node.js   │    │ PostgreSQL  │    │   Redis     │
│ Web Server  │───▶│ API Server  │───▶│  Database   │    │   Cache     │
│   :8080     │    │   :3000     │    │   :5432     │    │   :6379     │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

## Services

- **web**: Nginx web server serving the frontend
- **api**: Node.js API server with health checks
- **db**: PostgreSQL database with sample data
- **cache**: Redis cache for performance

## Quick Start

1. **Navigate to this directory**:
   ```bash
   cd examples/simple-web-app
   ```

2. **Start all services** (using Docker Compose MCP):
   Ask Claude: "Start all services in my Docker Compose project"

3. **Check service status**:
   Ask Claude: "Show me the status of all running containers"

4. **View the application**:
   Open http://localhost:8080 in your browser

## MCP Server Testing Commands

### Basic Operations
```
"Start all services in detached mode"
"Stop all services and remove containers"
"Show me the current status of all containers"
"Build all services without using cache"
```

### Monitoring and Logs
```
"Show me the logs from the API service"
"Get the last 50 lines of logs from all services"
"Start watching this project for file changes"
"Check the health of all services"
```

### Database Operations
```
"Execute 'SELECT * FROM users LIMIT 5' in the database"
"Run database migrations"
"Create a backup of the database"
"Reset the database and reload sample data"
```

### Testing and Development
```
"Run the test suite in the API service"
"Execute the health check script in the API container"
"Run a performance test on the API service"
```

### Advanced Operations
```
"Get performance metrics from the MCP server"
"Show me cache statistics and hit ratios"
"Start a watch session that rebuilds on changes"
"Stop all active watch sessions"
```

## Service Endpoints

### Web Interface
- **Frontend**: http://localhost:8080
- **Health**: http://localhost:8080/health

### API Endpoints
- **Health Check**: http://localhost:3000/health
- **Status**: http://localhost:3000/api/status
- **Database Test**: http://localhost:3000/api/database/test
- **Cache Test**: http://localhost:3000/api/cache/test
- **Performance**: http://localhost:3000/api/performance/heavy

### Database
- **Host**: localhost:5432
- **Database**: testdb
- **User**: testuser
- **Password**: testpass

### Cache
- **Host**: localhost:6379

## Expected MCP Behavior

### Context Reduction
The MCP server should filter verbose Docker output:
- ✅ **Filtered out**: Layer downloads, build cache, verbose pulls
- ✅ **Preserved**: Service status, errors, warnings, completion messages
- ✅ **Result**: 90%+ reduction in output size

### Tool Coverage
This example exercises all 14 MCP tools:
1. **compose_up** - Starting services
2. **compose_down** - Stopping services  
3. **compose_ps** - Listing containers
4. **compose_logs** - Viewing logs
5. **compose_build** - Building services
6. **compose_exec** - Running commands
7. **compose_test** - Running tests
8. **compose_watch_start** - File watching
9. **compose_watch_stop** - Stop watching
10. **compose_watch_status** - Watch status
11. **compose_migrate** - Database migrations
12. **compose_db_reset** - Database reset
13. **compose_db_backup** - Database backup
14. **compose_optimization** - Performance metrics

### Performance Features
- **Caching**: Configuration caching for faster operations
- **Parallel Execution**: Multi-service operations run concurrently
- **Session Management**: Long-running operations tracked properly
- **Metrics**: Filtering performance tracked and reported

## Troubleshooting

### Services Won't Start
```bash
# Check if Docker is running
docker info

# Check if ports are available
netstat -ln | grep -E ":(8080|3000|5432|6379)"

# Manual start for debugging
docker-compose up --no-deps web
```

### Database Connection Issues
```bash
# Check database logs
docker-compose logs db

# Test database connectivity
docker-compose exec db pg_isready -U testuser -d testdb
```

### API Health Issues
```bash
# Check API logs
docker-compose logs api

# Test API directly
curl http://localhost:3000/health
```

## Development

### Modifying the Application
1. Edit files in `html/`, `api/`, or `db/`
2. Ask Claude: "Rebuild the services that changed"
3. Ask Claude: "Start watching for file changes"

### Running Tests
Ask Claude: "Run the test suite in the API service"

### Database Operations
Ask Claude: "Execute 'SELECT COUNT(*) FROM users' in the database container"

## Clean Up

Ask Claude: "Stop all services and remove all containers, networks, and volumes"

This will completely clean up all resources created by this example.