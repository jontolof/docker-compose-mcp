-- Migration: Add user profiles
-- Version: 002
-- Description: Add user profiles table and additional user fields

-- Add profile fields to users table
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500);
ALTER TABLE users ADD COLUMN website_url VARCHAR(300);
ALTER TABLE users ADD COLUMN location VARCHAR(100);
ALTER TABLE users ADD COLUMN birth_date DATE;
ALTER TABLE users ADD COLUMN timezone VARCHAR(50) DEFAULT 'UTC';

-- Create user_profiles table for extended information
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    social_twitter VARCHAR(100),
    social_github VARCHAR(100),
    social_linkedin VARCHAR(100),
    newsletter_subscribed BOOLEAN DEFAULT false,
    email_notifications BOOLEAN DEFAULT true,
    privacy_level VARCHAR(20) DEFAULT 'public' CHECK (privacy_level IN ('public', 'friends', 'private')),
    last_login_at TIMESTAMP WITH TIME ZONE,
    login_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create trigger for user_profiles updated_at
CREATE TRIGGER update_user_profiles_updated_at 
    BEFORE UPDATE ON user_profiles 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create indexes
CREATE INDEX idx_user_profiles_privacy ON user_profiles(privacy_level);
CREATE INDEX idx_user_profiles_newsletter ON user_profiles(newsletter_subscribed);
CREATE INDEX idx_user_profiles_last_login ON user_profiles(last_login_at);

-- Create user_sessions table for session tracking
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(500) UNIQUE NOT NULL,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for sessions
CREATE INDEX idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_user_sessions_active ON user_sessions(is_active);
CREATE INDEX idx_user_sessions_expires ON user_sessions(expires_at);

-- Insert sample profile data for existing users
INSERT INTO user_profiles (user_id, social_github, social_twitter, newsletter_subscribed, email_notifications, privacy_level)
SELECT 
    id,
    CASE 
        WHEN username = 'johnsmith' THEN 'johnsmith-dev'
        WHEN username = 'janedoe' THEN 'jane-doe-dev'
        WHEN username = 'admin' THEN 'admin-user'
        ELSE NULL
    END,
    CASE 
        WHEN username = 'johnsmith' THEN 'johnsmith_dev'
        WHEN username = 'janedoe' THEN 'jane_doe_dev'
        ELSE NULL
    END,
    CASE 
        WHEN username = 'admin' THEN false
        ELSE true
    END,
    true,
    CASE 
        WHEN username = 'admin' THEN 'private'
        ELSE 'public'
    END
FROM users;

-- Update existing users with profile information
UPDATE users SET 
    bio = CASE 
        WHEN username = 'johnsmith' THEN 'Full-stack developer passionate about Docker and containerization.'
        WHEN username = 'janedoe' THEN 'Database specialist and migration expert.'
        WHEN username = 'admin' THEN 'System administrator'
        ELSE NULL
    END,
    location = CASE 
        WHEN username = 'johnsmith' THEN 'San Francisco, CA'
        WHEN username = 'janedoe' THEN 'New York, NY'
        WHEN username = 'admin' THEN 'Remote'
        ELSE NULL
    END,
    timezone = CASE 
        WHEN username = 'johnsmith' THEN 'America/Los_Angeles'
        WHEN username = 'janedoe' THEN 'America/New_York'
        WHEN username = 'admin' THEN 'UTC'
        ELSE 'UTC'
    END;

-- Create a view for user profiles with complete information
CREATE VIEW user_profiles_complete AS
SELECT 
    u.id,
    u.username,
    u.full_name,
    u.email,
    u.bio,
    u.avatar_url,
    u.website_url,
    u.location,
    u.timezone,
    u.is_active,
    u.created_at as user_created_at,
    u.updated_at as user_updated_at,
    p.social_twitter,
    p.social_github,
    p.social_linkedin,
    p.newsletter_subscribed,
    p.email_notifications,
    p.privacy_level,
    p.last_login_at,
    p.login_count,
    p.created_at as profile_created_at,
    p.updated_at as profile_updated_at,
    (SELECT COUNT(*) FROM posts WHERE author_id = u.id AND status = 'published') as published_posts_count,
    (SELECT COUNT(*) FROM comments WHERE author_id = u.id) as comments_count
FROM users u
LEFT JOIN user_profiles p ON u.id = p.user_id
WHERE u.is_active = true;