#!/bin/bash

if ! /usr/bin/systemctl is-active --quiet replay.service; then
    echo "Replay was stopped, restarting..."

    if service replay start; then
        echo "Restarted replay."
    else
        echo "Failed to restart replay."

        exit 1
    fi
fi

exit 0
