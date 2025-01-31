.PHONY: build maroon-redeploy cluster-start cluster-delete maroon-logs

build:
	docker build -t maroon:latest .

maroon-redeploy:
	kubectl delete -f deploy/maroon/maroon-deployment.yaml
	kubectl delete -f deploy/maroon/maroon-service.yaml
	docker build -t maroon:latest .
	kind load docker-image maroon:latest --name oltp-multi-region
	kubectl apply -f deploy/maroon/maroon-service.yaml
	kubectl apply -f deploy/maroon/maroon-deployment.yaml

cluster-start:
	kind create cluster --config deploy/cluster/kind-config.yaml

	kind load docker-image maroon:latest --name oltp-multi-region

	kubectl apply -f deploy/etcd/etcd-service.yaml
	kubectl apply -f deploy/etcd/etcd.yaml
	

	echo 'wait etcd-0'
	kubectl wait --for=condition=Ready pod/etcd-0
	echo 'wait etcd-1'
	kubectl wait --for=condition=Ready pod/etcd-1
	echo 'wait etcd-2'
	kubectl wait --for=condition=Ready pod/etcd-2

	echo 'etcd started'

	kubectl apply -f deploy/maroon/maroon-service.yaml
	sleep 1 # TODO: some weird behavior on DNS resolution
	kubectl apply -f deploy/maroon/maroon-deployment.yaml

cluster-delete:
	kind delete cluster --name oltp-multi-region

maroon-logs:
	kubectl logs -l app=maroon --follow --prefix