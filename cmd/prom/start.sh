#!/usr/bin/env sh

# shell is funny

construction "$@"
if [[ $? -eq 0 ]];then
  echo "start prometheus =============>"
  exec /bin/prometheus --config.file=/prometheus/prometheus.yml \
                       --storage.tsdb.path=/prometheus \
                       --web.console.libraries=/usr/share/prometheus/console_libraries \
                       --web.console.templates=/usr/share/prometheus/consoles
else
  echo "init prometheus configuration file error"
  exit $?
fi
