#!/bin/sh
set -e

# $1: deb -> "remove"/"purge"/"upgrade"; rpm -> "0" on erase, "1" on upgrade.
case "$1" in
	remove | purge | 0)
		systemctl disable --now deepcoolgo.service >/dev/null 2>&1 || true
		;;
esac
