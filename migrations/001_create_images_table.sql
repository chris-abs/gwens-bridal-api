CREATE TABLE IF NOT EXISTS images (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    s3_key VARCHAR(255) NOT NULL UNIQUE,
    s3_url VARCHAR(500) NOT NULL,
    category VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- Index for fast category lookups
CREATE INDEX idx_images_category ON images(category) WHERE is_active = true;

-- Insert some sample data
INSERT INTO images (filename, s3_key, s3_url, category) VALUES 
('sample-wedding-dress.jpg', 'images/sample-wedding-dress.jpg', 'https://your-bucket.s3.amazonaws.com/images/sample-wedding-dress.jpg', 'bridal'),
('sample-prom-dress.jpg', 'images/sample-prom-dress.jpg', 'https://your-bucket.s3.amazonaws.com/images/sample-prom-dress.jpg', 'prom')
ON CONFLICT (s3_key) DO NOTHING;