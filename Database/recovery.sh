#!/bin/bash

docker-compose up -d
docker-compose restart nginx_1
docker-compose restart nginx_2

