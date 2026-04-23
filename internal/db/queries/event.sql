-- name: CreateEvent :one
INSERT INTO events (
    name,
    ini_date,
    end_date,
    ini_time,
    end_time,
    location,
    budget,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetEvent :one
SELECT * FROM events
WHERE id = $1;

-- name: ListEvents :many
SELECT * FROM events
ORDER BY created_at DESC;

-- name: UpdateEvent :one
UPDATE events
SET name = $2,
    ini_date = $3,
    end_date = $4,
    ini_time = $5,
    end_time = $6,
    location = $7,
    budget = $8,
    status = $9
WHERE id = $1
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;