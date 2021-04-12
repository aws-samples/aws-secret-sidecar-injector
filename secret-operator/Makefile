# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
OIDC_PROVIDER=$(shell aws eks describe-cluster --name $$(kubectl config current-context | cut -d "/" -f 2) --query "cluster.identity.oidc.issuer" --output text | sed -e "s/^https:\/\///")

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
	
all: manager

test1:
	@echo ${IMG}

init:
	cp config/manager/kustomization_bkup.yaml config/manager/kustomization.yaml
	cp config/manager/manager_bkup.yaml config/manager/manager.yaml
	rm config/manager/*.bak

install:
	echo "Creating the AWS resources..."
	aws cloudformation create-stack --stack-name EKS-Secrets-Operator-Stack --template-body file://cfn.yaml --parameters ParameterKey=OIDCPROVIDER,ParameterValue=${OIDC_PROVIDER} --capabilities CAPABILITY_NAMED_IAM
	aws cloudformation wait stack-create-complete --stack-name EKS-Secrets-Operator-Stack
	echo "Stack creation complete, the output values are..."
	aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs"
	make set_aws_parameters

set_aws_parameters:
	echo "Preparing the controller manager manifest files..."
	$(info OPERATOR_REGION=$(shell aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs[?OutputKey=='Region'].OutputValue" --output text))
	$(info SQS_URL=$(shell aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs[?OutputKey=='QueueURL'].OutputValue" --output text))
	$(info IAM_ARN=$(shell aws cloudformation describe-stacks --stack-name EKS-Secrets-Operator-Stack --query "Stacks[0].Outputs[?OutputKey=='IAMRole'].OutputValue" --output text))
	sed -i .bak "s,OPERATOR_REGION,${OPERATOR_REGION},g" config/manager/manager.yaml
	sed -i .bak "s,SQS_URL,${SQS_URL},g" config/manager/manager.yaml
	sed -i .bak "s,IAM_ARN,${IAM_ARN},g" config/manager/manager.yaml
	make install_k8s

# Install CRDs into a cluster
install_k8s: manifests
	echo "Creating the required k8s resources..."
	kustomize build config/crd | kubectl apply -f -
	make deploy IMG=amazon/aws-secrets-manager-secret-rotator:v1

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	echo "Deploying the controller using the pre-built registry image..."
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Provisions a sample deployment and SecretsRotationMapping
test_operator:
	echo "Creating test deployment - operatortest..."
	kubectl create -f config/samples/awssecretsoperator_v1_secretsrotationmapping.yaml
	kubectl create -f config/samples/deployment.yaml

# Delete the k8s resources and the cloudformation stack resources	
delete:
	echo "Cleaning up the k8s and aws resources..."
	kubectl delete -f config/samples/deployment.yaml
	kubectl delete -f config/samples/awssecretsoperator_v1_secretsrotationmapping.yaml
	kustomize build config/default | kubectl delete -f -
	aws cloudformation delete-stack --stack-name EKS-Secrets-Operator-Stack
	aws cloudformation wait stack-delete-complete --stack-name EKS-Secrets-Operator-Stack
	make init

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go
	
# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
