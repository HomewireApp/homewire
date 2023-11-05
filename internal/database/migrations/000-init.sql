-- +migrate Up
CREATE TABLE "identities" (
  "id" char(36) primary key not null,
  "created_at" timestamp not null default current_timestamp,
  "name" varchar(32) not null unique,
  "private_key" blob not null,
  "displayed_name" varchar(128) not null
);

CREATE TABLE "wires" (
  "id" char(36) primary key not null,
  "created_at" timestamp not null default current_timestamp,
  "name" varchar(32) not null unique,
  "private_key" blob not null,
  "otp_secret" blob not null
);

-- +migrate Down
DROP TABLE "identities";
DROP TABLE "wires";
