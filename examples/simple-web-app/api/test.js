// Simple test suite for the API
const http = require('http');
const { execSync } = require('child_process');

const API_HOST = process.env.API_HOST || 'localhost';
const API_PORT = process.env.API_PORT || 3000;

console.log('🧪 Running API tests...');

// Test utilities
function makeRequest(path) {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: API_HOST,
      port: API_PORT,
      path: path,
      method: 'GET',
      timeout: 5000
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          resolve({ status: res.statusCode, data: parsed });
        } catch (e) {
          resolve({ status: res.statusCode, data: data });
        }
      });
    });

    req.on('error', reject);
    req.on('timeout', () => reject(new Error('Request timeout')));
    req.end();
  });
}

// Test cases
const tests = [
  {
    name: 'Health Check',
    path: '/health',
    expectedStatus: [200, 503], // 503 is OK if services are starting
    validate: (data) => data.status && data.services
  },
  {
    name: 'API Status',
    path: '/api/status',
    expectedStatus: 200,
    validate: (data) => data.service && data.version
  },
  {
    name: 'Database Test',
    path: '/api/database/test',
    expectedStatus: [200, 503],
    validate: (data) => data.success !== undefined
  },
  {
    name: 'Cache Test',
    path: '/api/cache/test',
    expectedStatus: [200, 503],
    validate: (data) => data.success !== undefined
  },
  {
    name: 'Performance Test',
    path: '/api/performance/heavy?iterations=10',
    expectedStatus: 200,
    validate: (data) => data.duration_ms && data.performance
  },
  {
    name: '404 Handler',
    path: '/nonexistent',
    expectedStatus: 404,
    validate: (data) => data.error === 'Not found'
  }
];

// Run tests
async function runTests() {
  let passed = 0;
  let failed = 0;
  
  console.log(`\n📋 Running ${tests.length} tests...\n`);
  
  for (const test of tests) {
    try {
      console.log(`🧪 ${test.name}...`);
      const result = await makeRequest(test.path);
      
      // Check status
      const statusOk = Array.isArray(test.expectedStatus) 
        ? test.expectedStatus.includes(result.status)
        : result.status === test.expectedStatus;
        
      if (!statusOk) {
        throw new Error(`Expected status ${test.expectedStatus}, got ${result.status}`);
      }
      
      // Check validation
      if (test.validate && !test.validate(result.data)) {
        throw new Error('Response validation failed');
      }
      
      console.log(`  ✅ PASSED (${result.status})`);
      passed++;
      
    } catch (error) {
      console.log(`  ❌ FAILED: ${error.message}`);
      failed++;
    }
  }
  
  console.log(`\n📊 Test Results:`);
  console.log(`  ✅ Passed: ${passed}`);
  console.log(`  ❌ Failed: ${failed}`);
  console.log(`  📈 Success Rate: ${Math.round(passed / tests.length * 100)}%`);
  
  if (failed > 0) {
    console.log(`\n⚠️  Some tests failed. This may be expected if services are still starting up.`);
    process.exit(1);
  } else {
    console.log(`\n🎉 All tests passed!`);
    process.exit(0);
  }
}

// Handle timeout
setTimeout(() => {
  console.error('\n⏰ Tests timed out after 30 seconds');
  process.exit(1);
}, 30000);

runTests().catch((error) => {
  console.error(`\n💥 Test runner failed: ${error.message}`);
  process.exit(1);
});