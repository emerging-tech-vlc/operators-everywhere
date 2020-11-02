# ETCD Operator Demo

In this demo we are going to show how you can use the ETCD operator in order to create an ETCD instance on K8s.

## Deploying the ETCD Operator

1. Create a namespace `operators-demos` for working with operators
2. Login into the OpenShift Console
2. Deploy the `ETCD Operator` using the in-cluster OperatorHub

## Inspecting the ETCD Operator

1. Take a look at the operator provided APIS

## Deploying an ETCD Instance

1. Create an `EtcdCluster` instance
2. Explore the different resources created by the operator

   1. Pods
   2. Services
3. Connect to the ETCD Cluster

   ~~~sh
   oc -n operators-demos run --rm -i --env="ALLOW_NONE_AUTHENTICATION=yes" --tty etcdclient --image bitnami/etcd:latest --restart=Never -- /bin/bash
   ~~~

   ~~~sh
   # Try to get a non-existing key
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 get foo
   # Create foo key
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 put foo bar
   # Get foo key
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 get foo
   # Get foo key value only
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 get foo --print-value-only
   # Get Etcd members
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 member list
   ~~~

## Scale up the ETCD cluster

1. Edit the `EtcdCluster` object and set replicas to `5`
2. Check the Etcd members

   ~~~sh
   oc -n operators-demos run --rm -i --env="ALLOW_NONE_AUTHENTICATION=yes" --tty etcdclient --image bitnami/etcd:latest --restart=Never -- /bin/bash
   ~~~

   ~~~sh
   # Get Etcd members
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 member list
   ~~~

## Remove a replica member from the cluster

1. Delete one etcd member

   ~~~sh
   oc -n operators-demos delete $(oc -n operators-demos get pods -l app=etcd -o name | head -1)
   ~~~
2. Check the members

   ~~~sh
   oc -n operators-demos run --rm -i --env="ALLOW_NONE_AUTHENTICATION=yes" --tty etcdclient --image bitnami/etcd:latest --restart=Never -- /bin/bash
   ~~~

   ~~~sh
   # Get Etcd members
   etcdctl --endpoints http://my-test-etcd-client.operators-demos.svc:2379 member list
   ~~~