#!/usr/bin/env bash

set -e

# Given a number as input, perform ROT-N...

# Default to ROT-13 if no input is given
N=${1:-13}

# Validate input is a number between 0 and 26
if ! [[ "$N" =~ ^[0-9]+$ ]] || [ "$N" -lt 0 ] || [ "$N" -gt 26 ]; then
  echo "Bad input. Please provide a number between 0 and 26."
  exit 1
fi

# 0 and 26 output the original
if [ "$N" -eq 0 ] || [ "$N" -eq 26 ]; then
  cat
  exit 0
fi

# Create the tr arguments for ROT-N
LOWER_DST=$(echo {a..z} | tr -d ' ' | cut -c $((N+1))-26)$(echo {a..z} | tr -d ' ' | cut -c 1-$N)
UPPER_DST=$(echo {A..Z} | tr -d ' ' | cut -c $((N+1))-26)$(echo {A..Z} | tr -d ' ' | cut -c 1-$N)

tr "A-Za-z" "$UPPER_DST$LOWER_DST"


