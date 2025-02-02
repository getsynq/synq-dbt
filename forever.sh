#!/usr/bin/env bash

echo "forever.sh pid: $$"
echo "forever.sh pgid: $(ps -o pgid= $$)"

trap "echo 'forever.sh received termination signal'; exit" SIGINT SIGTERM

while true; do
  echo "Press [CTRL+C] to stop.."
  sleep 1
done
