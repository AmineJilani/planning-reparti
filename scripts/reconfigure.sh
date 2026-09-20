#!/bin/bash

echo "Reconfiguration : suppression de B"

pkill -9 -f "cmd/net"
pkill -9 -f "/tmp/go-build"
pkill -9 -f "cat /tmp/out_"
pkill -9 -f "tee /tmp/in_"

rm -f /tmp/in_A /tmp/in_C /tmp/out_A /tmp/out_C

mkfifo /tmp/in_A /tmp/in_C
mkfifo /tmp/out_A /tmp/out_C

exec 3<>/tmp/in_A
exec 4<>/tmp/out_A
exec 5<>/tmp/in_C
exec 6<>/tmp/out_C

go run ./cmd/net -n A < /tmp/in_A > /tmp/out_A &
APP_A=$!

go run ./cmd/net -n C < /tmp/in_C > /tmp/out_C &
APP_C=$!

sleep 1

cat /tmp/out_A | tee /tmp/in_C &
PID_AC=$!

cat /tmp/out_C | tee /tmp/in_A &
PID_CA=$!

echo "Nouvelle topologie : A -> C -> A"
echo "APP_A=$APP_A APP_C=$APP_C"
echo "PID_AC=$PID_AC PID_CA=$PID_CA"

sleep 3600