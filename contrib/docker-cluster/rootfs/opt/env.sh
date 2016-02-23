#!/bin/bash -efu

export OPENSHIFT="${OPENSHIFT:-/data/src/github.com/openshift/origin}"
export OPENSHIFT_CONFIG_PATH="/opt/config"
export PATH="$OPENSHIFT/_output/local/bin/linux/amd64:$PATH"

CURL_CA_BUNDLE="$OPENSHIFT_CONFIG_PATH/openshift.local.config/master/ca.crt"
[ ! -f "$CURL_CA_BUNDLE" ] ||
	export CURL_CA_BUNDLE

KUBECONFIG="$OPENSHIFT_CONFIG_PATH/openshift.local.config/master/admin.kubeconfig"
[ ! -f "$KUBECONFIG" ] ||
	export KUBECONFIG
