-- Initialize database for Docker Compose MCP Demo
-- This script runs automatically when the database container starts

\echo 'Initializing Docker Compose MCP Demo database...'

-- Create tables for demo application
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO users (username, email) VALUES 
    ('demo_user', 'demo@example.com'),
    ('test_user', 'test@example.com'),
    ('mcp_user', 'mcp@example.com')
ON CONFLICT (username) DO NOTHING;

INSERT INTO posts (user_id, title, content) VALUES
    (1, 'Welcome to Docker Compose MCP Demo', 'This is a sample post to demonstrate the MCP server capabilities.'),
    (2, 'Testing Database Operations', 'This post shows how database operations work with the MCP server.'),
    (3, 'MCP Server Features', 'The MCP server provides 90%+ context reduction while maintaining full operational visibility.')
ON CONFLICT DO NOTHING;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at);

-- Create a view for posts with user information
CREATE OR REPLACE VIEW posts_with_users AS
SELECT 
    p.id,
    p.title,
    p.content,
    p.created_at,
    p.updated_at,
    u.username,
    u.email
FROM posts p
JOIN users u ON p.user_id = u.id
ORDER BY p.created_at DESC;

-- Migration tracking table (for testing migration tools)
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Record initial migration
INSERT INTO schema_migrations (version) VALUES ('001_initial_schema') 
ON CONFLICT (version) DO NOTHING;

\echo 'Database initialization completed successfully!'

-- Display summary
SELECT 
    'users' as table_name, 
    COUNT(*) as record_count 
FROM users
UNION ALL
SELECT 
    'posts' as table_name, 
    COUNT(*) as record_count 
FROM posts
UNION ALL
SELECT 
    'migrations' as table_name, 
    COUNT(*) as record_count 
FROM schema_migrations;