#!/usr/bin/env bash

SITE=${1:-D}
JOIN_TARGET=${2:-A}
PORT=${3:-4447}
rm -f /tmp/app_to_ctl_$SITE /tmp/ctl_to_app_$SITE
rm -f /tmp/ctl_to_net_$SITE /tmp/net_to_ctl_$SITE
rm -f /tmp/in_$SITE /tmp/out_$SITE

mkfifo /tmp/app_to_ctl_$SITE || exit 1
mkfifo /tmp/ctl_to_app_$SITE || exit 1
mkfifo /tmp/ctl_to_net_$SITE || exit 1
mkfifo /tmp/net_to_ctl_$SITE || exit 1
mkfifo /tmp/in_$SITE || exit 1
mkfifo /tmp/out_$SITE || exit 1

exec 50<>/tmp/app_to_ctl_$SITE
exec 51<>/tmp/ctl_to_app_$SITE
exec 52<>/tmp/ctl_to_net_$SITE
exec 53<>/tmp/net_to_ctl_$SITE
exec 54<>/tmp/in_$SITE
exec 55<>/tmp/out_$SITE

pkill -9 -f "cmd/app -n $SITE"
pkill -9 -f "cmd/ctl -n $SITE"
pkill -9 -f "cmd/net -n $SITE"
pkill -9 -f "app_to_ctl_$SITE"

PORT_PID=$(lsof -ti tcp:$PORT)
if [ -n "$PORT_PID" ]; then
  kill -9 $PORT_PID
fi


cat /tmp/app_to_ctl_$SITE > /tmp/net_to_ctl_$SITE &
CAT_PID=$!

go run ./cmd/net -n $SITE -join $JOIN_TARGET \
  -ctl-in /tmp/ctl_to_net_$SITE \
  -ctl-out /tmp/net_to_ctl_$SITE \
  < /tmp/in_$SITE > /tmp/out_$SITE &
NET_PID=$!

sleep 2

go run ./cmd/ctl -n $SITE -sites 20 \
  -net-out /tmp/ctl_to_net_$SITE \
  < /tmp/net_to_ctl_$SITE > /tmp/ctl_to_app_$SITE &
CTL_PID=$!

sleep 1

go run ./cmd/app -n $SITE \
  -addr 0.0.0.0 \
  -port $PORT \
  < /tmp/ctl_to_app_$SITE > /tmp/app_to_ctl_$SITE &
APP_PID=$!

echo "CAT_$SITE=$CAT_PID"
echo "NET_$SITE=$NET_PID"
echo "CTL_$SITE=$CTL_PID"
echo "APP_$SITE=$APP_PID"
echo "Interface : http://localhost:$PORT"