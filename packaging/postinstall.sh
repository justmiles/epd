#!/bin/sh
set -e

systemctl daemon-reload
systemctl enable epd.service
systemctl start epd.service
