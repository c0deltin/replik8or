# replik8or

A Kubernetes operator that replicates Secrets and ConfigMaps into namespaces.

## Deployment

### Helm Deployment
The preferred way to install replik8or is by using Helm:
```shell
$ helm upgrade --install replik8or oci://ghcr.io/c0deltin/helm-charts/replik8or
```
or 

```shell
$ helm repo add c0deltin https://c0deltin.github.io/helm-charts
$ helm repo update
$ helm upgrade --install replik8or c0deltin/replik8or
```

### Configuration

There are two ways of configuring ``replik8or``: Using environemnt variables or using flags.   
The following configuration values are available:

| env key                  | flag                     | default                | description                                                                        |
|--------------------------|--------------------------|------------------------|------------------------------------------------------------------------------------|
| `METRICS_ADDR`           | `metrics-addr`           | 0                      | Address under which the metrics server will be availabele. (_disabled by default_) |
| `HEALTH_PROBE_ADDR`      | `health-probe-addr`      | 0                      | Address under which the health probe will be available. (_disabled by default_)    |
| `DISALLOWED_NAMESPACES`  | `disallowed-namespaces`  |                        | Namespaces for which replicating resources is disabled. (_comma seperated_)        |


## Usage

Adding `replik8or.c0deltin.dev/replication-allowed="true"` to a ConfigMap or Secret will tell the operator to    
replicate this object to all other namespaces.    
To be more precise add ``replik8or.c0deltin.dev/desired-namespaces="<my-ns-1>,<another-ns>"``.
In this case the operator will only replicate the object into those namespaces.

> [!IMPORTANT]   
> `DISALLOWED_NAMESPACES` will always beat the `desired-namespaces` annotation.


## ToDo's
- [ ] Allow adding `desired-namespaces` annotation after replicas already have been created (remove replicas from namespaces not in annotation)