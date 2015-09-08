## bosh-init configuration

This CPI release is used in bosh-init integration tests.

Example configuration without registry:

```yaml
cloud_provider:
  release: bosh-warden-cpi
  mbus: https://admin:admin@10.244.0.42:6868
  properties:
    cpi:
      warden:
        connect_network: tcp
        connect_address: 0.0.0.0:7777
        network_pool: 10.244.0.0/16
        host_ip: 192.168.54.4
      agent:
        mbus: https://admin:admin@0.0.0.0:6868
        blobstore:
          provider: local
          options:
            blobstore_path: /var/vcap/micro_bosh/data/cache
```

Example configuration with registry:

```yaml
cloud_provider:
  release: bosh-warden-cpi
  ssh_tunnel:
    host: 10.244.0.42
    port: 22
    user: vcap
    password: c1oudc0w
  mbus: https://admin:admin@10.244.0.42:6868
  properties:
    cpi:
      warden:
        connect_network: tcp
        connect_address: 0.0.0.0:7777
        network_pool: 10.244.0.0/16
        host_ip: 192.168.54.4
      actions:
        agent_env_service: registry
      agent:
        mbus: https://admin:admin@0.0.0.0:6868
        blobstore:
          provider: local
          options:
            blobstore_path: /var/vcap/micro_bosh/data/cache
```
