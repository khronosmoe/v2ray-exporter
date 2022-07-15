# V2Ray Exporter

An exporter that collect V2Ray metrics over its [Stats API][stats-api] and export them to Prometheus.

- [V2Ray Exporter](#v2ray-exporter)
  - [Quick Start](#quick-start)
  - [Tutorial](#tutorial)
  - [Digging Deeper](#digging-deeper)
  - [Special Thanks](#special-thanks)

[stats-api]: https://www.v2ray.com/chapter_02/stats.html

## Quick Start

```bash
brew tap khronosmoe/homebrew-repo
brew install v2ray_exporter
brew services start v2ray_exporter
```

By default, the `v2ray_exporter` listens on port `9110` and fetches V2Ray data via port `10500`.

Run `/usr/local/opt/v2ray_exporter/bin/v2ray_exporter_brew_services` for temporary use.

## Tutorial

```bash
brew install grafana prometheus node_exporter v2ray v2ray_exporter

```

### Exporter

Edit `/usr/local/etc/v2ray_exporter.args`

```bash
--v2ray.endpoint=127.0.0.1:10500
--web.listen-address=localhost:9110
--web.config=/usr/local/etc/tls.yaml
```

Edit `/usr/local/etc/node_exporter.args`

```bash
--web.listen-address=localhost:9100
--web.config=/usr/local/etc/tls.yaml
```

### Prometheus

Edit `/usr/local/etc/prometheus.args`

```bash
--config.file=/usr/local/etc/prometheus.yml
--storage.tsdb.path=/usr/local/var/prometheus
--web.listen-address=localhost:9090
--web.config.file=/usr/local/etc/tls.yaml
```

### TLS

```bash
openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 -keyout server.key -out server.crt -subj "/C=US/CN=localhost"
chmod 400 server.key server.crt
htpasswd -nBC 12 '' | tr -d ':\n'
```

Edit `/usr/local/etc/tls.yaml`

```yaml
tls_server_config:
  cert_file: server.crt
  key_file: server.key
basic_auth_users:
  demo: $2y$12$pncygVzjqhZdF5sTNj5FyOTYv2PjPHVyu7b8Q5Zt3flddqwdxMP6e%
```

Edit `/usr/local/etc/prometheus.yml`

```yaml
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: "prometheus"
    scheme: https
    static_configs:
      - targets: ["localhost:9090"]
    tls_config:
      ca_file: server.crt
      insecure_skip_verify: true
    basic_auth:
      username: demo
      password: demo
  - job_name: "node_exporter"
    scheme: https
    static_configs:
      - targets: ["localhost:9100"]
    tls_config:
      ca_file: server.crt
      insecure_skip_verify: true
    basic_auth:
      username: demo
      password: demo
  - job_name: "v2ray_exporter"
    scheme: https
    static_configs:
      - targets: ["localhost:9110"]
    tls_config:
      ca_file: server.crt
      insecure_skip_verify: true
    basic_auth:
      username: demo
      password: demo
```

### V2Ray

Edit `/usr/local/etc/v2ray/config.json`

```json
{
  "stats": {},
  "api": {
    "tag": "api",
    "services": ["StatsService"]
  },
  "policy": {
    "levels": {
      "0": {
        "statsUserUplink": true,
        "statsUserDownlink": true
      }
    },
    "system": {
      "statsInboundUplink": true,
      "statsInboundDownlink": true,
      "statsOutboundUplink": true,
      "statsOutboundDownlink": true
    }
  },
  "inbounds": [
    {
      "tag": "tcp",
      "port": 3307,
      "protocol": "vmess",
      "settings": {
        "clients": [
          {
            "email": "auser",
            "id": "e731f153-4f31-49d3-9e8f-ff8f396135ef",
            "level": 0,
            "alterId": 64
          }
        ]
      }
    },
    {
      "tag": "api",
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      }
    }
  ],
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom",
      "settings": {}
    }
  ],
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "type": "field",
        "inboundTag": "api",
        "outboundTag": "api"
      }
    ]
  }
}
```

## Digging Deeper

The exporter doesn't retain the original metric names from V2Ray intentionally. You may find out why in the [comments][explaination-of-metric-names].

For users who do not really care about the internal changes, but only need a mapping table, here it is:

| Statistic Metric                           | Exposed Metric                                                               |
| :----------------------------------------- | :--------------------------------------------------------------------------- |
| `inbound>>>tag-name>>>traffic>>>uplink`    | `v2ray_traffic_uplink_bytes_total{dimension="inbound",target="tag-name"}`    |
| `inbound>>>tag-name>>>traffic>>>downlink`  | `v2ray_traffic_downlink_bytes_total{dimension="inbound",target="tag-name"}`  |
| `outbound>>>tag-name>>>traffic>>>uplink`   | `v2ray_traffic_uplink_bytes_total{dimension="outbound",target="tag-name"}`   |
| `outbound>>>tag-name>>>traffic>>>downlink` | `v2ray_traffic_downlink_bytes_total{dimension="outbound",target="tag-name"}` |
| `user>>>user-email>>traffic>>>uplink`      | `v2ray_traffic_uplink_bytes_total{dimension="user",target="user-email"}`     |
| `user>>>user-email>>>traffic>>>downlink`   | `v2ray_traffic_downlink_bytes_total{dimension="user",target="user-email"}`   |
| ...                                        | ...                                                                          |

## Special Thanks

- <https://github.com/wi1dcard/v2ray-exporter>
- <https://github.com/xmapst/v2ray-tracing>
- <https://github.com/oliver006/redis_exporter>
- <https://github.com/prometheus/exporter-toolkit/tree/master/web>
- <https://github.com/LeiShi1313/node_exporter/blob/master/collector/v2ray.go>
- <https://github.com/Homebrew/homebrew-core/blob/master/Formula/node_exporter.rb>
- <https://inuits.eu/blog/prometheus-tls>
