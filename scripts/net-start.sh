#!/bin/bash

# TESTER SEULEMENT NET
pkill -9 -f "cmd/net"
pkill -9 -f "cmd/router"
pkill -9 -f "/tmp/go-build"

rm -f /tmp/in_A /tmp/in_B /tmp/in_C /tmp/in_D /tmp/in_E
rm -f /tmp/out_A /tmp/out_B /tmp/out_C /tmp/out_D /tmp/out_E

rm -f /tmp/ctl_to_net_A /tmp/ctl_to_net_B /tmp/ctl_to_net_C
rm -f /tmp/net_to_ctl_A /tmp/net_to_ctl_B /tmp/net_to_ctl_C

mkfifo /tmp/in_A /tmp/in_B /tmp/in_C /tmp/in_D /tmp/in_E
mkfifo /tmp/out_A /tmp/out_B /tmp/out_C /tmp/out_D /tmp/out_E

mkfifo /tmp/ctl_to_net_A /tmp/ctl_to_net_B /tmp/ctl_to_net_C
mkfifo /tmp/net_to_ctl_A /tmp/net_to_ctl_B /tmp/net_to_ctl_C

exec 3<>/tmp/in_A
exec 4<>/tmp/out_A
exec 5<>/tmp/in_B
exec 6<>/tmp/out_B
exec 7<>/tmp/in_C
exec 8<>/tmp/out_C
exec 9<>/tmp/in_D
exec 10<>/tmp/out_D
exec 11<>/tmp/in_E
exec 12<>/tmp/out_E

exec 13<>/tmp/ctl_to_net_A
exec 14<>/tmp/net_to_ctl_A
exec 15<>/tmp/ctl_to_net_B
exec 16<>/tmp/net_to_ctl_B
exec 17<>/tmp/ctl_to_net_C
exec 18<>/tmp/net_to_ctl_C

go run ./cmd/router -sites A,B,C,D,E &
ROUTER=$!

sleep 1

go run ./cmd/net -n A -succ B -pred C -members A,B,C -ctl-in /tmp/ctl_to_net_A -ctl-out /tmp/net_to_ctl_A < /tmp/in_A > /tmp/out_A &
APP_A=$!

go run ./cmd/net -n B -succ C -pred A -members A,B,C -ctl-in /tmp/ctl_to_net_B -ctl-out /tmp/net_to_ctl_B < /tmp/in_B > /tmp/out_B &
APP_B=$!

go run ./cmd/net -n C -succ A -pred B -members A,B,C  -ctl-in /tmp/ctl_to_net_C -ctl-out /tmp/net_to_ctl_C < /tmp/in_C > /tmp/out_C &
APP_C=$!

sleep 5

echo "Test REGISTER terminé"
echo "ROUTER=$ROUTER APP_A=$APP_A APP_B=$APP_B APP_C=$APP_C"

sleep 3600