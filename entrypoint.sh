#!/bin/sh
set -e

ARGS=""

if [ -n "$INPUT_CONFIG" ]; then
    ARGS="$ARGS -config ${GITHUB_WORKSPACE}/${INPUT_CONFIG}"
else
    ARGS="$ARGS -content ${GITHUB_WORKSPACE}/${INPUT_SOURCE}"
    ARGS="$ARGS -title ${INPUT_TITLE}"
    if [ -n "$INPUT_STATIC" ]; then
        ARGS="$ARGS -static ${GITHUB_WORKSPACE}/${INPUT_STATIC}"
    fi
fi

ARGS="$ARGS -build ${GITHUB_WORKSPACE}/${INPUT_DESTINATION}"

echo "Running: pumice $ARGS"
pumice $ARGS
