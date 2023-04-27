#!/usr/bin/env bash

files=(listener.go store.go)

mkdir -p mock_rabbitmqstore
for file in "${files[@]}"; do
mockgen -source=$file > mock_rabbitmqstore/$file
done