# Monitoring

This directory contains sample configurations related to third party monitoring of Kache.

## Bootstrap

The `demo` folder contains a docker compose file and configurations to spin up a Kache server along 
with Prometheus and Grafana. This helps to get started with a instrumentd Kache environment and enables 
the development of Grafana dashboards.

To launch the containers:
`cd demo && docker compose up`

The output of `docker ps` should look like the following:

```bash
CONTAINER ID   IMAGE              COMMAND                  CREATED              STATUS             PORTS                                                                NAMES
adbd29616919   demo-kache         "./kache -config.fil…"   12 minutes ago       Up 4 minutes       0.0.0.0:80->80/tcp, 0.0.0.0:6728->6728/tcp, 0.0.0.0:6767->6767/tcp   demo-kache-1
9b32853b525b   redis:alpine       "docker-entrypoint.s…"   12 minutes ago       Up 4 minutes       0.0.0.0:6379->6379/tcp                                               demo-redis-1
10959c342b87   grafana/grafana    "/run.sh"                12 minutes ago       Up 4 minutes       0.0.0.0:3000->3000/tcp                                               demo-grafana-1
5d63219f1148   prom/prometheus    "/bin/prometheus --c…"   12 minutes ago       Up 4 minutes       0.0.0.0:9090->9090/tcp                                               demo-prometheus-1
```

When each container is up and running, the services can be accessed via:

- `localhost:8080`: kache listener
- `localhost:6767`: kache api
- `localhost:9090`: prometheus web ui
- `localhost:3000`: grafana (dashboards located at `/dashboards`)

## Dashboards

The `grafana/dashboards` folder contains a sample dashboard. Changes to the dashboard are made in the 
Grafana web interface. In order to save changes, click on the "Save Dashboard" icon in the menu 
in the upper right corner. Then, export the dashboard as a JSON file by clicking the "Save JSON 
to file" button. Finally, move this file to the "grafana/dashboards" folder.

## Cleanup

`CTRL+C` stops the running containers.  
`docker compose down` removes all data from previous runs.