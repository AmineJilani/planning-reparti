#!/usr/bin/env bash

sleep 1

echo "/=type=RESERVE_LOCAL/=id=R1/=room=SalleA/=date=2026-04-22/=start=10:00/=end=11:00/=user=A" > /tmp/in_A &

echo "/=type=RESERVE_LOCAL/=id=R2/=room=SalleA/=date=2026-04-22/=start=10:00/=end=11:00/=user=B" > /tmp/in_B &