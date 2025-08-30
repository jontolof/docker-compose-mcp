const express = require('express');
const { Client } = require('pg');
const redis = require('redis');

const app = express();
const port = process.env.PORT || 3000;

// Database configuration
const dbConfig = {
  host: process.env.DB_HOST || 'db',
  port: process.env.DB_PORT || 5432,
  database: process.env.DB_NAME || 'testdb',
  user: process.env.DB_USER || 'testuser',
  password: process.env.DB_PASSWORD || 'testpass',
};

// Redis configuration
const redisClient = redis.createClient({
  host: process.env.REDIS_HOST || 'cache',
  port: process.env.REDIS_PORT || 6379,
});

let dbClient;
let isDbConnected = false;
let isRedisConnected = false;

// Initialize database connection
async function initDatabase() {
  try {
    dbClient = new Client(dbConfig);
    await dbClient.connect();
    
    // Test the connection
    await dbClient.query('SELECT NOW()');
    isDbConnected = true;
    console.log('✅ Database connected successfully');
  } catch (error) {
    console.error('❌ Database connection failed:', error.message);
    isDbConnected = false;
  }
}

// Initialize Redis connection
async function initRedis() {
  try {
    await redisClient.connect();
    await redisClient.ping();
    isRedisConnected = true;
    console.log('✅ Redis connected successfully');
  } catch (error) {
    console.error('❌ Redis connection failed:', error.message);
    isRedisConnected = false;
  }
}

// Middleware
app.use(express.json());
app.use((req, res, next) => {
  console.log(`${new Date().toISOString()} - ${req.method} ${req.path}`);
  next();
});

// Health check endpoint
app.get('/health', async (req, res) => {
  const health = {
    status: 'healthy',
    timestamp: new Date().toISOString(),
    services: {
      api: 'healthy',
      database: isDbConnected ? 'healthy' : 'unhealthy',
      cache: isRedisConnected ? 'healthy' : 'unhealthy'
    }
  };
  
  const overallHealthy = isDbConnected && isRedisConnected;
  res.status(overallHealthy ? 200 : 503).json(health);
});

// API endpoints
app.get('/api/status', (req, res) => {
  res.json({
    service: 'Docker Compose MCP Demo API',
    version: '1.0.0',
    environment: process.env.NODE_ENV || 'development',
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    connections: {
      database: isDbConnected,
      cache: isRedisConnected
    }
  });
});

app.get('/api/database/test', async (req, res) => {
  if (!isDbConnected) {
    return res.status(503).json({ error: 'Database not connected' });
  }
  
  try {
    const result = await dbClient.query('SELECT NOW() as current_time, version() as version');
    res.json({
      success: true,
      data: result.rows[0]
    });
  } catch (error) {
    console.error('Database query error:', error);
    res.status(500).json({ error: error.message });
  }
});

app.get('/api/cache/test', async (req, res) => {
  if (!isRedisConnected) {
    return res.status(503).json({ error: 'Redis not connected' });
  }
  
  try {
    const testKey = 'mcp_test';
    const testValue = `test_${Date.now()}`;
    
    await redisClient.set(testKey, testValue, { EX: 60 });
    const retrieved = await redisClient.get(testKey);
    
    res.json({
      success: true,
      data: {
        written: testValue,
        retrieved: retrieved,
        match: testValue === retrieved
      }
    });
  } catch (error) {
    console.error('Redis operation error:', error);
    res.status(500).json({ error: error.message });
  }
});

// Performance endpoint for MCP optimization testing
app.get('/api/performance/heavy', (req, res) => {
  const iterations = parseInt(req.query.iterations) || 1000;
  const startTime = Date.now();
  
  // Simulate heavy computation
  let result = 0;
  for (let i = 0; i < iterations * 1000; i++) {
    result += Math.sqrt(i);
  }
  
  const endTime = Date.now();
  
  res.json({
    iterations: iterations * 1000,
    result: Math.floor(result),
    duration_ms: endTime - startTime,
    performance: {
      operations_per_second: Math.floor((iterations * 1000) / (endTime - startTime) * 1000),
      memory_usage: process.memoryUsage()
    }
  });
});

// Error handling middleware
app.use((error, req, res, next) => {
  console.error('API Error:', error);
  res.status(500).json({
    error: 'Internal server error',
    message: process.env.NODE_ENV === 'development' ? error.message : 'Something went wrong'
  });
});

// 404 handler
app.use('*', (req, res) => {
  res.status(404).json({
    error: 'Not found',
    path: req.originalUrl,
    available_endpoints: [
      '/health',
      '/api/status',
      '/api/database/test',
      '/api/cache/test',
      '/api/performance/heavy'
    ]
  });
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('🛑 SIGTERM received, shutting down gracefully');
  
  if (dbClient) {
    await dbClient.end();
    console.log('📊 Database connection closed');
  }
  
  if (redisClient) {
    await redisClient.quit();
    console.log('⚡ Redis connection closed');
  }
  
  process.exit(0);
});

// Start server
async function startServer() {
  console.log('🚀 Starting Docker Compose MCP Demo API...');
  
  // Initialize connections
  await Promise.all([
    initDatabase(),
    initRedis()
  ]);
  
  app.listen(port, '0.0.0.0', () => {
    console.log(`🌟 API server running on port ${port}`);
    console.log(`📋 Environment: ${process.env.NODE_ENV || 'development'}`);
    console.log(`🔗 Health check: http://localhost:${port}/health`);
  });
}

startServer().catch(console.error);