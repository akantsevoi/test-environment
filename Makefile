.PHONY: build redeploy-maroon start-cluser delete-cluster work-logs

build:
	docker build -t maroon:latest .

redeploy-maroon:
	kubectl delete -f deploy/maroon/maroon-deployment.yaml
	docker build -t maroon:latest .
	kind load docker-image maroon:latest --name oltp-multi-region
	kubectl apply -f deploy/maroon/maroon-deployment.yaml

start-cluster:
	kind create cluster --config deploy/cluster/kind-config.yaml

	kind load docker-image maroon:latest --name oltp-multi-region

	kubectl apply -f deploy/etcd/etcd.yaml
	kubectl apply -f deploy/etcd/etcd-service.yaml

	echo 'wait etcd-0'
	kubectl wait --for=condition=Ready pod/etcd-0
	echo 'wait etcd-1'
	kubectl wait --for=condition=Ready pod/etcd-1
	echo 'wait etcd-2'
	kubectl wait --for=condition=Ready pod/etcd-2

	echo 'etcd started'

	kubectl apply -f deploy/maroon/maroon-deployment.yaml

delete-cluster:
	kind delete cluster --name oltp-multi-region

maroon-logs:
	kubectl logs -l app=maroon --follow --prefix