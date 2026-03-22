#!/bin/sh

if [ -d /run/systemd/system ]; then
    systemctl disable --now gospeed-server || true
fi
