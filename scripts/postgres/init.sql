DO
$$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_roles WHERE rolname = 'pflow'
    ) THEN
        CREATE ROLE pflow WITH LOGIN PASSWORD 'pflow';
    END IF;
END
$$;

\connect postgres

SELECT 'CREATE DATABASE pflow OWNER pflow'
WHERE NOT EXISTS (
    SELECT 1 FROM pg_database WHERE datname = 'pflow'
)\gexec
