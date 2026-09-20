#!/bin/bash

(
    sleep 0.1
    go run ./cmd/net -n D -join A < /tmp/in_D > /tmp/out_D
) &

(
    sleep 0.1
    go run ./cmd/net -n E -join B < /tmp/in_E > /tmp/out_E
) &

wait

