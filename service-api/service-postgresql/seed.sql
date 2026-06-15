INSERT INTO assignees (name)
VALUES
    ('Ana Souza'),
    ('Bruno Lima'),
    ('Carla Mendes')
ON CONFLICT (name) DO NOTHING;

INSERT INTO tickets (
    title,
    description,
    requester_name,
    priority,
    status,
    assignee_id,
    assignment_mode,
    opened_at
)
SELECT
    'Impressora do financeiro sem conexão',
    'A impressora principal do setor financeiro não aparece nos computadores desde o início da manhã.',
    'Mariana Alves',
    'high',
    'open',
    a.id,
    'manual',
    NOW() - INTERVAL '45 minutes'
FROM assignees a
WHERE a.name = 'Ana Souza'
  AND NOT EXISTS (
      SELECT 1 FROM tickets WHERE title = 'Impressora do financeiro sem conexão'
  );

INSERT INTO tickets (
    title,
    description,
    requester_name,
    priority,
    status,
    assignee_id,
    assignment_mode,
    opened_at
)
SELECT
    'Solicitação de cadeira ergonômica',
    'A cadeira atual está com o encosto quebrado e precisa ser substituída.',
    'Paulo Ribeiro',
    'medium',
    'in_progress',
    a.id,
    'automatic',
    NOW() - INTERVAL '3 hours'
FROM assignees a
WHERE a.name = 'Bruno Lima'
  AND NOT EXISTS (
      SELECT 1 FROM tickets WHERE title = 'Solicitação de cadeira ergonômica'
  );

