#!/usr/bin/env sh

set -o errexit

if set -o | grep pipefail >/dev/null; then
  set -o pipefail
fi;

if [ -x /usr/local/bin/bash ]; then
	exec /usr/local/bin/bash -c "$@"
elif [ -x /usr/bin/bash ]; then
	exec /usr/bin/bash -c "$@"
elif [ -x /bin/bash ]; then
	exec /bin/bash -c "$@"
elif [ -x /usr/local/bin/sh ]; then
	exec /usr/local/bin/sh -c "$@"
elif [ -x /usr/bin/sh ]; then
	exec /usr/bin/sh -c "$@"
elif [ -x /bin/sh ]; then
	exec /bin/sh -c "$@"
else
	echo shell not found
	exit 1
fi
