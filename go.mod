module github.com/ceyhunalp/calypso_experiments

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/stretchr/testify v1.5.1
	go.dedis.ch/cothority/v3 v3.3.2
	go.dedis.ch/kyber/v3 v3.0.12
	go.dedis.ch/onet/v3 v3.2.1
	go.dedis.ch/protobuf v1.0.11
	go.etcd.io/bbolt v1.3.4
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898
	google.golang.org/appengine v1.6.5
	gopkg.in/urfave/cli.v1 v1.20.0
)

replace go.dedis.ch/onet/v3 => ../onet

//replace go.dedis.ch/cothority/v3 => /Users/alp/go-workspace/src/github.com/dedis/cothority
