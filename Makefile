.PHONY: build maroon-redeploy cluster-start cluster-delete maroon-logs test-kill-restore gen

install-tools:
	# TODO: fix it for other platforms https://grpc.io/docs/protoc-installation/
	# we need protoc tool
	brew install protobuf

	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest	

	go install go.uber.org/mock/mockgen

gen:
	mkdir -p gen
	protoc \
		--go_out=gen \
		--go_opt=paths=source_relative \
    	--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
    	proto/maroon/p2p/v1/maroon.proto 

	mockgen -source=internal/maroon/interface.go -destination=internal/maroon/mocks/interface_mock.go -package=mocks
	mockgen -source=internal/p2p/interface.go -destination=internal/p2p/mocks/interface_mock.go -package=mocks

build:
	# worker container
	docker build -t maroon:latest .

build-test:
	# test scripts
	go build -o bin/test-node-failure scripts/test/node-failure/main.go

test-kill-restore: build-test
	./bin/test-node-failure

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

cluster-add-delays:
	for node in $$(docker ps --filter "name=oltp-multi-region-work*" --format "{{.Names}}"); do \
    	echo "Adding delay to $$node"; \
    	docker exec "$$node" tc qdisc add dev eth0 root netem delay 50ms; \
	done

cluster-remove-delays:
	for node in $$(docker ps --filter "name=oltp-multi-region-work*" --format "{{.Names}}"); do \
		echo "Removing delay from $$node"; \
		docker exec "$$node" tc qdisc del dev eth0 root || true; \
	done