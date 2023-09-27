#!/bin/bash

service ssh start
service apache2 start
/usr/local/bin/docker-entrypoint.sh mysqld
