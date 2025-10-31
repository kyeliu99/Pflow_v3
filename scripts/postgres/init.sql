DO
$$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pflow_gateway') THEN
        CREATE ROLE pflow_gateway WITH LOGIN PASSWORD 'pflow_gateway';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pflow_form') THEN
        CREATE ROLE pflow_form WITH LOGIN PASSWORD 'pflow_form';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pflow_identity') THEN
        CREATE ROLE pflow_identity WITH LOGIN PASSWORD 'pflow_identity';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pflow_ticket') THEN
        CREATE ROLE pflow_ticket WITH LOGIN PASSWORD 'pflow_ticket';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pflow_workflow') THEN
        CREATE ROLE pflow_workflow WITH LOGIN PASSWORD 'pflow_workflow';
    END IF;
END
$$;

\connect postgres

SELECT 'CREATE DATABASE pflow_gateway OWNER pflow_gateway' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'pflow_gateway')\gexec;
SELECT 'CREATE DATABASE pflow_form OWNER pflow_form' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'pflow_form')\gexec;
SELECT 'CREATE DATABASE pflow_identity OWNER pflow_identity' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'pflow_identity')\gexec;
SELECT 'CREATE DATABASE pflow_ticket OWNER pflow_ticket' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'pflow_ticket')\gexec;
SELECT 'CREATE DATABASE pflow_workflow OWNER pflow_workflow' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = 'pflow_workflow')\gexec;
