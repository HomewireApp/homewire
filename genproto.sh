#!/usr/bin/env bash
set -e

protoc -I=./ --go_out=./ ./internal/proto/pb/homewire.proto