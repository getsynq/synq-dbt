#!/usr/bin/env bash

echo "forever.sh pid: $$"
echo "forever.sh pgid: $(ps -o pgid= $$)"

while true; do
  echo "Press [CTRL+C] to stop.."
  sleep 1
done