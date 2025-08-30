-- Migration: Initial schema
-- Version: 001
-- Description: Create initial tables for demo application

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email CITEXT UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    slug VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Posts table
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    published_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    author_name VARCHAR(100) NOT NULL,
    author_email CITEXT NOT NULL,
    content TEXT NOT NULL,
    is_approved BOOLEAN DEFAULT false,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_active ON users(is_active);

CREATE INDEX idx_posts_author ON posts(author_id);
CREATE INDEX idx_posts_category ON posts(category_id);
CREATE INDEX idx_posts_status ON posts(status);
CREATE INDEX idx_posts_published ON posts(published_at);
CREATE INDEX idx_posts_slug ON posts(slug);

CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_author ON comments(author_id);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_comments_approved ON comments(is_approved);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_posts_updated_at 
    BEFORE UPDATE ON posts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data
INSERT INTO categories (name, description, slug) VALUES
    ('Technology', 'Posts about technology and programming', 'technology'),
    ('Lifestyle', 'Posts about lifestyle and personal experiences', 'lifestyle'),
    ('Travel', 'Travel experiences and guides', 'travel'),
    ('Food', 'Recipes and food experiences', 'food');

INSERT INTO users (email, username, full_name, password_hash) VALUES
    ('john@example.com', 'johnsmith', 'John Smith', '$2a$10$example_hash_1'),
    ('jane@example.com', 'janedoe', 'Jane Doe', '$2a$10$example_hash_2'),
    ('admin@example.com', 'admin', 'Admin User', '$2a$10$example_hash_3');

-- Get category and user IDs for posts
WITH tech_cat AS (SELECT id FROM categories WHERE slug = 'technology'),
     lifestyle_cat AS (SELECT id FROM categories WHERE slug = 'lifestyle'),
     john_user AS (SELECT id FROM users WHERE username = 'johnsmith'),
     jane_user AS (SELECT id FROM users WHERE username = 'janedoe')

INSERT INTO posts (title, slug, content, excerpt, author_id, category_id, status, published_at) VALUES
    (
        'Getting Started with Docker Compose',
        'getting-started-docker-compose',
        'Docker Compose is a powerful tool for defining and running multi-container Docker applications. In this post, we''ll explore how to use it effectively.',
        'Learn the basics of Docker Compose and multi-container applications.',
        (SELECT id FROM john_user),
        (SELECT id FROM tech_cat),
        'published',
        CURRENT_TIMESTAMP - INTERVAL '2 days'
    ),
    (
        'Database Migrations Best Practices',
        'database-migrations-best-practices',
        'Managing database schema changes can be challenging. Here are some best practices for handling migrations in production environments.',
        'Essential tips for managing database migrations safely.',
        (SELECT id FROM jane_user),
        (SELECT id FROM tech_cat),
        'published',
        CURRENT_TIMESTAMP - INTERVAL '1 day'
    ),
    (
        'Work-Life Balance in Tech',
        'work-life-balance-tech',
        'Maintaining a healthy work-life balance is crucial for long-term success in the technology industry.',
        'Tips for maintaining balance in a fast-paced industry.',
        (SELECT id FROM john_user),
        (SELECT id FROM lifestyle_cat),
        'published',
        CURRENT_TIMESTAMP - INTERVAL '3 hours'
    );

-- Add some comments
WITH docker_post AS (SELECT id FROM posts WHERE slug = 'getting-started-docker-compose'),
     migration_post AS (SELECT id FROM posts WHERE slug = 'database-migrations-best-practices'),
     jane_user AS (SELECT id FROM users WHERE username = 'janedoe')

INSERT INTO comments (post_id, author_id, author_name, author_email, content, is_approved) VALUES
    (
        (SELECT id FROM docker_post),
        (SELECT id FROM jane_user),
        'Jane Doe',
        'jane@example.com',
        'Great introduction to Docker Compose! This helped me understand the basics.',
        true
    ),
    (
        (SELECT id FROM migration_post),
        NULL,
        'Anonymous User',
        'user@example.com',
        'Thanks for sharing these migration practices. Very helpful for our team.',
        true
    );

-- Create a view for published posts with author information
CREATE VIEW published_posts_view AS
SELECT 
    p.id,
    p.title,
    p.slug,
    p.content,
    p.excerpt,
    p.published_at,
    p.created_at,
    p.updated_at,
    u.username as author_username,
    u.full_name as author_name,
    c.name as category_name,
    c.slug as category_slug,
    (SELECT COUNT(*) FROM comments WHERE post_id = p.id AND is_approved = true) as comment_count
FROM posts p
JOIN users u ON p.author_id = u.id
LEFT JOIN categories c ON p.category_id = c.id
WHERE p.status = 'published'
ORDER BY p.published_at DESC;