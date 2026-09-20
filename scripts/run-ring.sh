#!/usr/bin/env bash

rm -f /tmp/in_A /tmp/in_B /tmp/in_C

mkfifo /tmp/in_A /tmp/in_B /tmp/in_C

go run ./cmd/ctl -n A -sites 3 < /tmp/in_A > /tmp/in_B &
go run ./cmd/ctl -n B -sites 3 < /tmp/in_B > /tmp/in_C &
go run ./cmd/ctl -n C -sites 3 < /tmp/in_C > /tmp/in_A &

echo "anneau de contrôle prêt : A -> B -> C -> A"

sleep 3600