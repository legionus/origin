#!/bin/bash -efu

. /opt/env.sh

docker_pidfile=/var/run/docker.pid
docker_logfile=/var/log/docker.log

docker daemon \
	--pidfile="$docker_pidfile" \
	--host=unix:///var/run/docker.sock \
	--exec-opt native.cgroupdriver=cgroupfs \
	--storage-driver=vfs \
	&>"$docker_logfile" &

docker_pid=
for i in 1 2 3 4 5 6 7 8 9 10; do
	sleep 0.2
	if [ -s "$docker_pidfile" ]; then
		read docker_pid < "$docker_pidfile" ||:
		[ -n "$docker_pid" ] &&
			kill -0 "$docker_pid" 2>/dev/null ||
			docker_pid=
		break
	fi
	printf 'INFO: waiting for docker ...\n'
done

if [ -z "$docker_pid" ]; then
	printf 'ERROR: docker does not start\n'
	exit 1
fi
printf 'INFO: docker started (pid=%s) logs are placed "%s"\n' "$docker_pid" "$docker_logfile"
