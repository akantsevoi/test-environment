## Purpose

Test environment for maroon migrator

## Install

`go install sigs.k8s.io/kind@v0.26.0`

## Run
`make build`  
`make cluster-start`
`make test-kill-restore` # will find and kill leader-node

`make cluster-delete`

## Troubleshooting:
Checks etcd status:
`kubectl exec etcd-0 -- etcdctl endpoint status --endpoints=http://etcd-0.etcd:2379,http://etcd-1.etcd:2379,http://etcd-2.etcd:2379 -w table`

DNS maroon:
`kubectl exec -it maroon-0 -- nslookup maroon-0.maroon.default.svc.cluster.local`

## Maroon-application

`cmd/app/main.go`