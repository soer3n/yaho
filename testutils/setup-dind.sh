apk add --no-cache git make musl-dev go curl
export GOROOT=/usr/lib/go
export GOPATH=/go
export PATH=/go/bin:$PATH
mkdir -p ${GOPATH}/src ${GOPATH}/bin
go get -u github.com/Masterminds/glide/...
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
chmod +x kubectl
export GOBIN=$(go env GOPATH)/bin
export PATH=$PATH:$GOBIN
mkdir -p $GOBIN
mv kubectl $GOBIN
wget https://github.com/kubernetes-sigs/kind/releases/download/v0.11.1/kind-linux-amd64 && chmod +x kind-linux-amd64 && mv kind-linux-amd64 $GOBIN/kind
git clone https://github.com/containernetworking/plugins.git
cd plugins
sh build_linux.sh
