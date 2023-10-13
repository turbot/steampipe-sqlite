-- Create Table To Hold Config
CREATE TABLE aws_config(config TEXT);
-- Configuration
INSERT INTO aws_config(config) VALUES ('{"profile":"silverwater", "regions":["*"]}');

-- Temp Configuration HCL
INSERT INTO aws_config(config) VALUES ('profile = "silverwater"');
INSERT INTO aws_config(config) VALUES ('profile = "aaa"');