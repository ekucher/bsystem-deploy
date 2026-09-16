-- The E2E stack only needs the Integration Core schema. authentik and the HUB
-- are replaced by mocks, so their databases are deliberately absent.
CREATE DATABASE bsystem_integration;
