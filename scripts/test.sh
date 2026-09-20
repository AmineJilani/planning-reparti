#!/usr/bin/env bash

rm -f /tmp/app_to_ctl_A /tmp/app_to_ctl_B /tmp/app_to_ctl_C
rm -f /tmp/ctl_to_app_A /tmp/ctl_to_app_B /tmp/ctl_to_app_C
rm -f /tmp/in_A /tmp/in_B /tmp/in_C

mkfifo /tmp/app_to_ctl_A /tmp/app_to_ctl_B /tmp/app_to_ctl_C
mkfifo /tmp/ctl_to_app_A /tmp/ctl_to_app_B /tmp/ctl_to_app_C
mkfifo /tmp/in_A /tmp/in_B /tmp/in_C

# éviter blocages FIFO
exec 3<>/tmp/app_to_ctl_A
exec 4<>/tmp/app_to_ctl_B
exec 5<>/tmp/app_to_ctl_C

exec 6<>/tmp/ctl_to_app_A
exec 7<>/tmp/ctl_to_app_B
exec 8<>/tmp/ctl_to_app_C

exec 9<>/tmp/in_A
exec 10<>/tmp/in_B
exec 11<>/tmp/in_C

# APP web pour chaque site
go run ./cmd/app -n A -addr 0.0.0.0 -port 4444 -net-out < /tmp/ctl_to_app_A > /tmp/app_to_ctl_A &
go run ./cmd/app -n B -addr 0.0.0.0 -port 4445 -net-out < /tmp/ctl_to_app_B > /tmp/app_to_ctl_B &
go run ./cmd/app -n C -addr 0.0.0.0 -port 4446 -net-out < /tmp/ctl_to_app_C > /tmp/app_to_ctl_C &

# CTL A
# reçoit depuis APP A + depuis C
# envoie vers APP A + vers B
{ cat /tmp/app_to_ctl_A & cat /tmp/in_A & } \
| go run ./cmd/ctl -n A -sites 3 \
| tee /tmp/ctl_to_app_A > /tmp/in_B &

# CTL B
# reçoit depuis APP B + depuis A
# envoie vers APP B + vers C
{ cat /tmp/app_to_ctl_B & cat /tmp/in_B & } \
| go run ./cmd/ctl -n B -sites 3 \
| tee /tmp/ctl_to_app_B > /tmp/in_C &

# CTL C
# reçoit depuis APP C + depuis B
# envoie vers APP C + vers A
{ cat /tmp/app_to_ctl_C & cat /tmp/in_C & } \
| go run ./cmd/ctl -n C -sites 3 \
| tee /tmp/ctl_to_app_C > /tmp/in_A &

echo "Interfaces web prêtes :"
echo "Site A : http://127.0.0.1:4444"
echo "Site B : http://127.0.0.1:4445"
echo "Site C : http://127.0.0.1:4446"

wait