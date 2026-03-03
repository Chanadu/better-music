#!/usr/bin/env sh

migrate -path migrations -database $DATABASE_URL down
migrate -path migrations -database $DATABASE_URL up
