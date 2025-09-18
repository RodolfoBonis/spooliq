-- Script para criar múltiplos bancos de dados
-- Este script é executado automaticamente quando o PostgreSQL é iniciado pela primeira vez

-- Criar banco para o Keycloak
CREATE DATABASE keycloak_db;
GRANT ALL PRIVILEGES ON DATABASE keycloak_db TO user;

-- O banco da aplicação (spooliq_db) já é criado pela variável POSTGRES_DB 