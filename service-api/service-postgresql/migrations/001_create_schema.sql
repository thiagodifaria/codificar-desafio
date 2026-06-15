CREATE TABLE IF NOT EXISTS assignees (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL UNIQUE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    last_assigned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tickets (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(160) NOT NULL,
    description TEXT NOT NULL,
    requester_name VARCHAR(120) NOT NULL,
    priority VARCHAR(16) NOT NULL
        CHECK (priority IN ('low', 'medium', 'high')),
    status VARCHAR(24) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    assignee_id BIGINT NOT NULL REFERENCES assignees(id),
    assignment_mode VARCHAR(16) NOT NULL
        CHECK (assignment_mode IN ('manual', 'automatic')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tickets_resolution_consistency CHECK (
        (status IN ('resolved', 'closed') AND resolved_at IS NOT NULL)
        OR
        (status IN ('open', 'in_progress') AND resolved_at IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee_status
    ON tickets(assignee_id, status);
CREATE INDEX IF NOT EXISTS idx_tickets_opened_at ON tickets(opened_at DESC);

