-- Create databases for different services
CREATE DATABASE userdb;
CREATE DATABASE orderdb;
CREATE DATABASE productdb;
CREATE DATABASE paymentdb;
CREATE DATABASE notificationdb;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE userdb TO postgres;
GRANT ALL PRIVILEGES ON DATABASE orderdb TO postgres;
GRANT ALL PRIVILEGES ON DATABASE productdb TO postgres;
GRANT ALL PRIVILEGES ON DATABASE paymentdb TO postgres;
GRANT ALL PRIVILEGES ON DATABASE notificationdb TO postgres;