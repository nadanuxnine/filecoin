#!/bin/bash

LOG_FILE="$1"
CSV_FILE="$2"

echo "miner,epoch,param.penalty,param.gasreward,param.wincount,block_reward,total_reward,penalty,reward_payable" > "$CSV_FILE"

cat "$LOG_FILE" | rg "FRRIST" | rg "AwardBlockReward" | jq ".msg" | awk '{ print $5 $7 $9 $11 $13 $15 $17 $19 $21}' | cut -d '"' -f 1 >> "$CSV_FILE"
