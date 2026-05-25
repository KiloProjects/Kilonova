#!/bin/bash

SESSION_NAME="kilonova"

# Start tmux session only if it doesn't already exist
tmux has-session -t $SESSION_NAME 2>/dev/null

if [ $? != 0 ]; then
    tmux new-session -d -s $SESSION_NAME
    tmux send-keys -t $SESSION_NAME "cd /home/alexv/kilonova" C-m
    tmux send-keys -t $SESSION_NAME "./runkn.sh --css" C-m
fi
