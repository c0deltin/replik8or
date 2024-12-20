# replik8or
A Kubernetes operator that replicates ConfigMap's and Secrets.

## Description
**replik8or** copies annotated ConfigMap's and Secrets into other Kubernetes namespaces.
The project is heavily inspired by [emberstack/kubernetes-reflector](https://github.com/emberstack/kubernetes-reflector)! The reason I've rebuilt the operator is that I wanted a solution written in Go.

## Deployment

### Configuration

The following helm values are available (`replik8or.envs`):
- ``disallowedNamespaces`` (optional) define namespaces (comma-seperated) that should be excluded from the operator.

## Usage

Adding ``replik8or.c0deltin.io/reflection-allowed="true"`` to a ConfigMap or Secret will tell the operator to    
replicate this object to all other namespaces.   
To be more precise ``replik8or.c0deltin.io/allowed-namespaces="<my-ns-1>,<another-ns>"`` can be added. 
In this case the operator will only replicate the object into those namespaces.

> [!IMPORTANT]   
> The ``disallowedNamespaces`` option will always beat any of the configured namespaces in the `allowed-namespaces` annotation. 