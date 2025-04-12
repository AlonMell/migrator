CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем связь категорий с постами
ALTER TABLE posts
ADD COLUMN category_id UUID,
ADD CONSTRAINT fk_posts_category FOREIGN KEY (category_id) REFERENCES categories (id);
