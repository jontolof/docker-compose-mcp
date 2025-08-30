# Workflow Integration Examples

This guide shows how to use the Docker Compose MCP Server with Claude for different development workflows.

## Development Workflow

### Starting a New Project

**You:** "I have a new Docker Compose project. Help me get it set up and running."

**Expected MCP Actions:**
1. `compose_ps` - Check current state
2. `compose_build` - Build services if needed
3. `compose_up` - Start services in detached mode
4. `compose_logs` - Show startup logs

**Conversation Example:**
```
You: I have a new Docker Compose project. Help me get it set up and running.

Claude: I'll help you set up your Docker Compose project. Let me start by checking the current state and then get everything running.

[Uses compose_ps to check current containers]
[Uses compose_build to build the services] 
[Uses compose_up with detach mode]
[Uses compose_logs to show startup status]

Great! I've successfully started your Docker Compose project. Here's what happened:
- Built 3 services (web, api, database)
- Started all services in detached mode  
- All health checks are passing
- Services are accessible on their configured ports

Your application should now be running at http://localhost:8080
```

### Daily Development Loop

**You:** "Start my development environment and watch for changes."

**Expected MCP Actions:**
1. `compose_up` - Start services
2. `compose_watch_start` - Begin file watching
3. `compose_ps` - Verify services are healthy

**Conversation Example:**
```
You: Start my development environment and watch for changes.

Claude: I'll get your development environment running and set up file watching for automatic rebuilds.

[Uses compose_up to start services]
[Uses compose_watch_start to begin watching]
[Uses compose_ps to verify status]

Perfect! Your development environment is ready:
- All services started successfully
- File watcher active (Session ID: watch_abc123)
- Auto-rebuild enabled for code changes
- Health checks passing for all services

I'll automatically rebuild and restart services when you modify files in your project.
```

### Testing Workflow

**You:** "Run all tests and show me the results."

**Expected MCP Actions:**
1. `compose_test` - Run test suites
2. `compose_logs` - Get detailed test output if needed
3. `compose_ps` - Check service states

**Conversation Example:**
```
You: Run all tests and show me the results.

Claude: I'll run your test suite and give you a comprehensive report.

[Uses compose_test for each service with tests]
[Uses compose_logs if tests fail to get details]

Test Results Summary:
✅ API Service Tests: 47/47 passed (Coverage: 89.2%)
✅ Web Service Tests: 23/23 passed  
✅ Integration Tests: 12/12 passed
⏱️ Total execution time: 2m 34s

All tests are passing! Your code is ready for deployment.
```

## Database Workflow

### Database Migration

**You:** "I need to run database migrations and make sure everything is working."

**Expected MCP Actions:**
1. `compose_migrate` - Run migrations
2. `compose_exec` - Verify database state
3. `compose_db_backup` - Create backup after migration

**Conversation Example:**
```
You: I need to run database migrations and make sure everything is working.

Claude: I'll handle your database migrations safely, including a backup.

[Uses compose_db_backup to create pre-migration backup]
[Uses compose_migrate to run migrations up]
[Uses compose_exec to verify schema changes]

Database Migration Complete:
✅ Backup created: backup-pre-migration-20240115-143022
✅ Applied 3 new migrations (v001 → v004)
✅ Schema validation successful
✅ All foreign key constraints intact

Your database is now up to date with the latest schema changes.
```

### Database Reset for Development

**You:** "Reset the database to clean state with sample data."

**Expected MCP Actions:**
1. `compose_db_backup` - Create backup first
2. `compose_db_reset` - Reset database
3. `compose_exec` - Verify reset worked
4. `compose_logs` - Check for any errors

**Conversation Example:**
```
You: Reset the database to clean state with sample data.

Claude: I'll reset your development database safely with a backup first.

[Uses compose_db_backup to create safety backup]
[Uses compose_db_reset with confirmation and seed data]
[Uses compose_exec to verify the reset]

Database Reset Complete:
✅ Safety backup created: backup-20240115-143045
✅ Database dropped and recreated
✅ Schema migrations applied (v001 → v004)
✅ Sample data loaded successfully
✅ 3 users, 15 posts, 8 comments created

Your development database is now in a clean state with fresh sample data.
```

## Production Deployment Workflow

### Pre-Deployment Checks

**You:** "Check if my application is ready for production deployment."

**Expected MCP Actions:**
1. `compose_build` - Build production images
2. `compose_test` - Run full test suite  
3. `compose_up` - Start in production mode
4. `compose_exec` - Run health checks
5. `compose_optimization` - Get performance metrics

**Conversation Example:**
```
You: Check if my application is ready for production deployment.

Claude: I'll run comprehensive pre-deployment checks to ensure your application is production-ready.

[Uses compose_build with no-cache for fresh builds]
[Uses compose_test for full test coverage]
[Uses compose_up with production environment]
[Uses compose_exec for health checks and smoke tests]
[Uses compose_optimization for performance metrics]

Production Readiness Report:
✅ Clean builds completed (no cache issues)
✅ All tests passing (Coverage: 92.3%)  
✅ Production containers healthy
✅ Database migrations up to date
✅ External service connectivity verified
✅ Performance metrics within acceptable ranges
✅ Security checks passed
✅ Resource usage optimized

Your application is ready for production deployment!
```

## Debugging Workflow

### Investigating Issues

**You:** "My API service is failing. Help me debug the issue."

**Expected MCP Actions:**
1. `compose_ps` - Check service status
2. `compose_logs` - Get error logs
3. `compose_exec` - Interactive debugging
4. `compose_optimization` - Check performance issues

**Conversation Example:**
```
You: My API service is failing. Help me debug the issue.

Claude: I'll help you diagnose the API service issues systematically.

[Uses compose_ps to check service status]
[Uses compose_logs api --tail 100 to get recent errors]
[Uses compose_exec to check internal service health]

Debug Analysis:
❌ API service: Unhealthy (Exit code: 1)
🔍 Error found: Database connection timeout
📊 Root cause: Database service not responding

Let me check the database service:
[Uses compose_logs database to check DB logs]
[Uses compose_exec database to test connectivity]

Resolution:
✅ Database was out of memory
✅ Restarted database service  
✅ API reconnected successfully
✅ All services now healthy

The issue was database memory exhaustion. I've restarted the services and everything is working normally now.
```

### Performance Investigation

**You:** "The application seems slow. Can you help me identify performance bottlenecks?"

**Expected MCP Actions:**
1. `compose_optimization` - Get performance metrics
2. `compose_logs` - Check for performance warnings
3. `compose_exec` - Run performance diagnostics
4. `compose_ps` - Check resource usage

**Conversation Example:**
```
You: The application seems slow. Can you help me identify performance bottlenecks?

Claude: I'll analyze your application's performance and identify bottlenecks.

[Uses compose_optimization to get detailed metrics]
[Uses compose_logs to scan for performance warnings]
[Uses compose_exec to run internal diagnostics]

Performance Analysis:
📊 Context Reduction: 94.2% (Excellent)
⚡ Cache Hit Rate: 67% (Good)
🐌 Slow Queries Detected: 3 database queries >2s
💾 Memory Usage: API service at 89% (High)
🔄 Connection Pool: Database connections near limit

Recommendations:
1. Add indexes for slow queries (I can help with migrations)
2. Increase API service memory limit
3. Optimize database connection pooling
4. Consider adding Redis caching layer

Would you like me to help implement any of these optimizations?
```

## Monitoring Workflow

### Continuous Monitoring

**You:** "Start monitoring my services and alert me if anything goes wrong."

**Expected MCP Actions:**
1. `compose_watch_start` - Begin monitoring
2. `compose_ps` - Initial health check
3. `compose_optimization` - Set baseline metrics

**Conversation Example:**
```
You: Start monitoring my services and alert me if anything goes wrong.

Claude: I'll set up continuous monitoring for your Docker Compose services.

[Uses compose_watch_start for change detection]
[Uses compose_ps to establish baseline health]
[Uses compose_optimization to record baseline metrics]

Monitoring Active:
🔍 File watcher: Session watch_monitor_789
📊 Health checks: Every 30 seconds
⚡ Performance tracking: Enabled
📈 Baseline metrics recorded

I'm now monitoring:
- Service health and availability
- Resource usage patterns  
- Performance metrics
- File system changes
- Log anomalies

I'll automatically alert you if I detect:
- Service failures or crashes
- Performance degradation >20%
- Error rate increases
- Resource usage spikes
- Security-related log entries

Your services are now under active monitoring.
```

## Team Collaboration Workflow

### Environment Sync

**You:** "A teammate made changes. Help me sync my environment with the latest updates."

**Expected MCP Actions:**
1. `compose_down` - Stop current services
2. `compose_build` - Rebuild with latest code
3. `compose_migrate` - Apply any new migrations
4. `compose_up` - Start updated services
5. `compose_test` - Verify everything works

**Conversation Example:**
```
You: A teammate made changes. Help me sync my environment with the latest updates.

Claude: I'll help you sync with your teammate's changes safely.

[Uses compose_down to stop current services cleanly]
[Uses compose_build --no-cache to rebuild everything fresh]
[Uses compose_migrate to apply any new database changes]
[Uses compose_up to start the updated environment]
[Uses compose_test to verify the changes work]

Environment Sync Complete:
✅ Stopped previous environment cleanly
✅ Pulled latest code changes
✅ Rebuilt 4 services with updates
✅ Applied 2 new database migrations
✅ Started updated services successfully  
✅ All tests passing with new changes

Your environment is now synced with your teammate's latest changes and ready for development.
```

## Advanced Workflow Patterns

### Multi-Environment Management

**You:** "Switch from development to staging configuration."

**Expected MCP Actions:**
1. `compose_down` - Stop dev environment
2. `compose_build` - Build staging images  
3. `compose_up` - Start with staging config
4. `compose_exec` - Verify staging settings

### Blue-Green Deployment Testing

**You:** "Test a blue-green deployment scenario."

**Expected MCP Actions:**
1. `compose_up` - Start blue environment
2. `compose_build` - Build green version
3. `compose_up` - Start green alongside blue
4. `compose_exec` - Run health checks on both
5. `compose_down` - Switch traffic simulation

### Disaster Recovery Testing

**You:** "Test our backup and recovery procedures."

**Expected MCP Actions:**
1. `compose_db_backup` - Create test backup
2. `compose_db_reset` - Simulate data loss
3. `compose_db_backup` - Restore from backup  
4. `compose_exec` - Verify data integrity

## Best Practices

### Effective Communication with Claude

1. **Be Specific**: "Start the web service" vs "Start all services"
2. **Include Context**: "I'm debugging a login issue" helps Claude prioritize relevant checks
3. **Confirm Destructive Operations**: Claude will ask for confirmation before database resets
4. **Use Natural Language**: "Show me what's broken" works better than technical jargon

### Workflow Optimization

1. **Chain Operations**: "Build, test, and deploy if everything passes"
2. **Set Expectations**: "This might take a few minutes" helps Claude manage timeouts  
3. **Monitor Progress**: Ask for status updates on long operations
4. **Save States**: Use backups before major changes

### Error Recovery

1. **Stay Calm**: Claude will help systematically diagnose issues
2. **Provide Context**: Share what you were doing when problems started
3. **Follow Suggestions**: Claude's recommendations are based on best practices
4. **Learn from Issues**: Ask Claude to explain what went wrong and how to prevent it

This workflow integration makes Docker Compose operations feel natural and conversational while maintaining the power and precision of the underlying tools.