#!/bin/sh

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    systemctl enable --now gospeed
fi