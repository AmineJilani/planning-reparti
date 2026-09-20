#!/usr/bin/env bash

SITE=${1:-D}

pkill -9 -f "net -n $SITE"
pkill -9 -f "ctl -n $SITE"
pkill -9 -f "app -n $SITE"
pkill -9 -f "cmd/net -n $SITE"
pkill -9 -f "cmd/ctl -n $SITE"
pkill -9 -f "cmd/app -n $SITE"
pkill -9 -f "cat /tmp/app_to_ctl_$SITE"

echo "Site $SITE crashé brutalement"