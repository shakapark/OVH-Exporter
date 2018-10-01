# OVH-Exporter

## Docker Install

~~~ shell
docker run -d --name ovh-exporter -v "<path/to/config/file>:/app/config/ovh.yml" -p "<ip>:<port>:9147" shakapark/ovh-exporter:tag
~~~

## Prometheus

```yml
scrape_configs:
  - job_name: 'ovh'
    metrics_path: /ovh
    static_configs:
    - targets: ['<ip>:<port>']
```