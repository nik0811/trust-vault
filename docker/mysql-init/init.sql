CREATE DATABASE IF NOT EXISTS datahub CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
USE datahub;

CREATE TABLE IF NOT EXISTS metadata_aspect_v2 (
  urn VARCHAR(500) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  aspect VARCHAR(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  version BIGINT NOT NULL,
  metadata LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  systemmetadata LONGTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
  createdon DATETIME(6) NOT NULL,
  createdby VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  createdfor VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
  PRIMARY KEY (urn, aspect, version),
  INDEX urnIndex (urn)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
