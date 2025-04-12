CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID NOT NULL,
    title VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id)
);
