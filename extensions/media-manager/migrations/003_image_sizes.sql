CREATE TABLE IF NOT EXISTS media_image_sizes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    mode VARCHAR(20) NOT NULL DEFAULT 'fit',
    source VARCHAR(100) NOT NULL DEFAULT 'default',
    quality INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Default sizes
INSERT INTO media_image_sizes (name, width, height, mode, source) VALUES
    ('thumbnail', 150, 150, 'crop', 'default'),
    ('medium', 250, 250, 'fit', 'default'),
    ('large', 500, 500, 'fit', 'default')
ON CONFLICT (name) DO NOTHING;
