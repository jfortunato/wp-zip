#!/bin/bash

service ssh start
/usr/local/bin/docker-entrypoint.sh apache2-foreground
