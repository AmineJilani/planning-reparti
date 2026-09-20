#!/usr/bin/env bash

# APP + CTL + NET + ROUTER


BASE_SITES="A B C"
# Nettoyage ancien lancement


pkill -9 cat
pkill -9 -f "cmd/app"
pkill -9 -f "cmd/ctl"
pkill -9 -f "cmd/net"
pkill -9 -f "cmd/router"
pkill -9 -f "/tmp/go-build"


rm -f /tmp/in_* /tmp/out_*
rm -f /tmp/ctl_to_net_* /tmp/net_to_ctl_*
rm -f /tmp/app_to_ctl_* /tmp/ctl_to_app_*


# Création des FIFOs pour tous les sites possibles
for SITE in $BASE_SITES; do
  rm -f /tmp/app_to_ctl_$SITE /tmp/ctl_to_app_$SITE
  rm -f /tmp/ctl_to_net_$SITE /tmp/net_to_ctl_$SITE
  rm -f /tmp/in_$SITE /tmp/out_$SITE


  mkfifo /tmp/app_to_ctl_$SITE || exit 1
  mkfifo /tmp/ctl_to_app_$SITE || exit 1
  mkfifo /tmp/ctl_to_net_$SITE || exit 1
  mkfifo /tmp/net_to_ctl_$SITE || exit 1
  mkfifo /tmp/in_$SITE || exit 1
  mkfifo /tmp/out_$SITE || exit 1
done


# Éviter les blocages FIFO
FD=3
for SITE in $BASE_SITES; do
  eval "exec ${FD}<>/tmp/app_to_ctl_$SITE"; FD=$((FD+1))
  eval "exec ${FD}<>/tmp/ctl_to_app_$SITE"; FD=$((FD+1))
  eval "exec ${FD}<>/tmp/ctl_to_net_$SITE"; FD=$((FD+1))
  eval "exec ${FD}<>/tmp/net_to_ctl_$SITE"; FD=$((FD+1))
  eval "exec ${FD}<>/tmp/in_$SITE"; FD=$((FD+1))
  eval "exec ${FD}<>/tmp/out_$SITE"; FD=$((FD+1))
done


# ROUTER : connaît tous les sites possibles
go run ./cmd/router &
ROUTER=$!


sleep 1


# NET initiaux A/B/C
go run ./cmd/net -n A -succ B -pred C -members A,B,C \
  -ctl-in /tmp/ctl_to_net_A \
  -ctl-out /tmp/net_to_ctl_A \
  < /tmp/in_A > /tmp/out_A &
NET_A=$!


go run ./cmd/net -n B -succ C -pred A -members A,B,C \
  -ctl-in /tmp/ctl_to_net_B \
  -ctl-out /tmp/net_to_ctl_B \
  < /tmp/in_B > /tmp/out_B &
NET_B=$!


go run ./cmd/net -n C -succ A -pred B -members A,B,C \
  -ctl-in /tmp/ctl_to_net_C \
  -ctl-out /tmp/net_to_ctl_C \
  < /tmp/in_C > /tmp/out_C &
NET_C=$!


sleep 2


# APP -> CTL pour A/B/C
cat /tmp/app_to_ctl_A > /tmp/net_to_ctl_A &
cat /tmp/app_to_ctl_B > /tmp/net_to_ctl_B &
cat /tmp/app_to_ctl_C > /tmp/net_to_ctl_C &


# APP web A/B/C
go run ./cmd/app -n A -addr 0.0.0.0 -port 4444 \
  < /tmp/ctl_to_app_A > /tmp/app_to_ctl_A &
APP_A=$!


go run ./cmd/app -n B -addr 0.0.0.0 -port 4445 \
  < /tmp/ctl_to_app_B > /tmp/app_to_ctl_B &
APP_B=$!


go run ./cmd/app -n C -addr 0.0.0.0 -port 4446 \
  < /tmp/ctl_to_app_C > /tmp/app_to_ctl_C &
APP_C=$!


# CTL A/B/C
go run ./cmd/ctl -n A -sites 3 -net-out /tmp/ctl_to_net_A \
  < /tmp/net_to_ctl_A > /tmp/ctl_to_app_A &
CTL_A=$!


go run ./cmd/ctl -n B -sites 3 -net-out /tmp/ctl_to_net_B \
  < /tmp/net_to_ctl_B > /tmp/ctl_to_app_B &
CTL_B=$!


go run ./cmd/ctl -n C -sites 3 -net-out /tmp/ctl_to_net_C \
  < /tmp/net_to_ctl_C > /tmp/ctl_to_app_C &
CTL_C=$!


echo ""
echo "Architecture lancée : APP <-> CTL <-> NET <-> ROUTER"
echo "Site A : http://127.0.0.1:4444"
echo "Site B : http://127.0.0.1:4445"
echo "Site C : http://127.0.0.1:4446"
echo ""
echo "Sites dynamiques  :"
echo "Ajouter un site : ./scripts/add_site.sh NOUVEAUSITE SITEPRESENT PORT"


echo ""
echo "ROUTER=$ROUTER"
echo "NET_A=$NET_A NET_B=$NET_B NET_C=$NET_C"
echo "CTL_A=$CTL_A CTL_B=$CTL_B CTL_C=$CTL_C"
echo "APP_A=$APP_A APP_B=$APP_B APP_C=$APP_C"


wait
