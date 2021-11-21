module github.com/dh1tw/touchctl

go 1.16

// replace github.com/dh1tw/streamdeck => /Users/tobias/go/src/github.com/dh1tw/streamdeck
// replace github.com/dh1tw/remoteSwitch => /Users/tobias/go/src/github.com/dh1tw/remoteSwitch
// replace github.com/dh1tw/remoteRotator => /Users/tobias/go/src/github.com/dh1tw/remoteRotator
// replace github.com/dh1tw/hid => /Users/tobias/go/src/github.com/dh1tw/hid
// replace github.com/dh1tw/streamdeck-buttons => /Users/tobias/go/src/github.com/dh1tw/streamdeck-buttons

require (
	github.com/asim/go-micro/plugins/broker/nats/v3 v3.0.0-20210416163442-a91d1f7a3dbb
	github.com/asim/go-micro/plugins/registry/nats/v3 v3.0.0-20210416163442-a91d1f7a3dbb
	github.com/asim/go-micro/plugins/transport/nats/v3 v3.0.0-20210416163442-a91d1f7a3dbb
	github.com/asim/go-micro/v3 v3.5.2
	github.com/dh1tw/remoteRotator v0.6.3-0.20210910212249-1d525322f67c
	github.com/dh1tw/remoteSwitch v0.2.2-0.20210910212220-2ebfcf967620
	github.com/dh1tw/streamdeck v0.1.5
	github.com/dh1tw/streamdeck-buttons v0.3.0
	github.com/nats-io/nats.go v1.13.0
)

replace github.com/gogo/protobuf v0.0.0-20190410021324-65acae22fc9 => github.com/gogo/protobuf v0.0.0-20190723190241-65acae22fc9d
