#!/bin/bash

docker-compose up -d
docker exec mysql_slave sh -c "nginx restart"