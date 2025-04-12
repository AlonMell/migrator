CREATE TABLE logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    level VARCHAR(10) NOT NULL,
    message TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Добавляем несколько тестовых записей
INSERT INTO
    logs (level, message)
VALUES
    ('INFO', 'Система инициализирована'),
    ('INFO', 'Базовая миграция применена');
