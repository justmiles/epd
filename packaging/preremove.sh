#!/bin/sh
set -e

systemctl stop epd.service || true
systemctl disable epd.service || true
