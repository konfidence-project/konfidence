#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
./setup-local-registry-common.sh "${SCRIPT_DIR}/ocm/component-constructor.yaml"
