# replik8or
A Kubernetes operator that replicates ConfigMap and Secrets.

## Description
replik8or copies annotated ConfigMap's and Secrets into other kubernetes namespaces.
It's heavily inspired by emberstack/kubernetes-reflector! The reason for this project is that I've once faced a bug in 
the kubernetes-reflector project but I really really don't like C# 🙃.

## Deployment

### Configuration

The following helm values are available (`replik8or.envs`):
- ``disallowedNamespaces`` (optional) define namespaces (coma-seperated) that should be excluded from the operator.

## Usage

Adding ``replik8or.c0deltin.io/reflection-allowed="true"`` to a ConfigMap or Secret will tell the operator to    
replicate this object to all other namespaces.   
To be more precise ``replik8or.c0deltin.io/allowed-namespaces="<my-ns-1>,<another-ns>"`` can be added. 
In this case the operator will only replicate the object into those namespaces.

> [!IMPORTANT]   
> The ``DISALLOWED_NAMESPACES`` will always beats any of the configured namespaces in the `allowed-namespaces` annotation. 