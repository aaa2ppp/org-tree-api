-- +goose Up
CREATE TABLE department (
    id SERIAL PRIMARY KEY,
    parent_id INTEGER NOT NULL REFERENCES department (id) ON DELETE RESTRICT,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(parent_id, name)
);

-- Technical node for top-level departments. DO NOT DELETE.
INSERT INTO department (id, parent_id, name) VALUES (-1, -1, '__virtual_root__');

CREATE TABLE employee (
    id SERIAL PRIMARY KEY,
    department_id INTEGER NOT NULL REFERENCES department (id) ON DELETE RESTRICT,
    full_name VARCHAR(200) NOT NULL,
    position VARCHAR(200) NOT NULL,
    hired_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE employee;
DROP TABLE department;

