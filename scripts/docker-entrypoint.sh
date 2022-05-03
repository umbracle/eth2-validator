#!/bin/sh

if [ "${1:0:1}" = '-' ]; then
    set -- beacon "$@"
fi

# Look for beacon subcommands.
if [ "$1" = 'server' ]; then
    shift
    set -- beacon server "$@"
fi

exec "$@"