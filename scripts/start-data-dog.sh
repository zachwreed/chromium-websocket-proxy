docker run -d --name dd-agent --cgroupns host \
              --pid host \
              -v /var/run/docker.sock:/var/run/docker.sock:ro \
              -v /proc/:/host/proc/:ro \
              -v /sys/fs/cgroup/:/host/sys/fs/cgroup:ro \
              -e DD_API_KEY="$DD_API_KEY" \
              -e DD_SITE=us5.datadoghq.com \
              -e DD_DOGSTATSD_NON_LOCAL_TRAFFIC="true"\
              -p 8125:8125/udp \
              gcr.io/datadoghq/agent:latest
