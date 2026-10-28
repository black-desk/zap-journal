#!/bin/sh
set -eu
find_tool() {
	tool_name=$1
	shift
	# Prefer system binaries over wrappers that may alter journalctl output.
	for path in "$@"; do
		if [ -x "$path" ]; then
			printf '%s\n' "$path"
			return
		fi
	done
	command -v "$tool_name" || { echo "$tool_name is unavailable" >&2; return 1; }
}
journald=$(find_tool systemd-journald /usr/lib/systemd/systemd-journald /lib/systemd/systemd-journald)
journalctl=$(find_tool journalctl /usr/bin/journalctl /bin/journalctl)
if [ "${1:-}" = --read ]; then
	shift
	work=$1
	shift
	exec "$journalctl" --directory="$work/journal" --no-pager --all --quiet -o json "$@"
fi
if [ "${1:-}" != --isolated ]; then
	if [ "${1:-}" = --check ]; then
		shift
		for tool in unshare mount cat touch readlink dirname rm; do
			command -v "$tool" >/dev/null || { echo "$tool is unavailable" >&2; exit 1; }
		done
		exec unshare --user --map-root-user --mount --pid --fork --kill-child --mount-proc \
			sh "$0" --isolated --check "$@"
	fi
	work=$1
	# Clean up from outside the mount namespace, after its processes exit.
	trap 'rm -rf -- "$work"' EXIT
	unshare --user --map-root-user --mount --pid --fork --kill-child --mount-proc \
		sh "$0" --isolated "$@"
	exit 0
fi
shift
check=false
if [ "${1:-}" = --check ]; then
	check=true
	shift
fi
# Check the isolation before making any mounts, even if invoked manually.
[ "$$" -eq 1 ] || { echo 'Expected an isolated PID namespace' >&2; exit 1; }
work=$1
config=$(dirname -- "$(readlink -f -- "$0")")/journald.conf
mount --make-rprivate /
mount -t tmpfs tmpfs /run
mount -t tmpfs tmpfs /var/log
mount -t tmpfs tmpfs /etc/systemd
machine_id=$(cat /etc/machine-id)
mkdir -p /run/systemd/journal.zaptest "/run/log/journal/$machine_id.zaptest"
mount --bind "$work/socket" /run/systemd/journal.zaptest
mount --bind "$work/journal" "/run/log/journal/$machine_id.zaptest"
touch /etc/systemd/journald@zaptest.conf
mount --bind "$config" /etc/systemd/journald@zaptest.conf
mount -o remount,bind,ro /etc/systemd/journald@zaptest.conf
if "$check"; then
	exit 0
fi
"$journald" zaptest &
pid=$!
trap 'kill -TERM "$pid" 2>/dev/null || :; wait "$pid" || :' EXIT
# The Go parent holds stdin open for the duration of the test.
read -r ignored || :
