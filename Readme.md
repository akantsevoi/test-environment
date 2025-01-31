## Purpose

Test environment for maroon migrator

## Install

`go install sigs.k8s.io/kind@v0.26.0`

## Run
`make build`
`make cluster-start`

`make cluster-delete`

## Troubleshooting:
Checks etcd status:
`kubectl exec etcd-0 -- etcdctl endpoint status --endpoints=http://etcd-0.etcd:2379,http://etcd-1.etcd:2379,http://etcd-2.etcd:2379 -w table`