module github.com/dh1tw/touchctl

go 1.17

// replace github.com/dh1tw/streamdeck => /Users/tobias/go/src/github.com/dh1tw/streamdeck
// replace github.com/dh1tw/streamdeck-buttons => /Users/tobias/go/src/github.com/dh1tw/streamdeck-buttons
// replace github.com/dh1tw/remoteSwitch => /Users/tobias/go/src/github.com/dh1tw/remoteSwitch
// replace github.com/dh1tw/remoteRotator => /Users/tobias/go/src/github.com/dh1tw/remoteRotator
replace github.com/dh1tw/hid => /Users/tobias/go/src/github.com/dh1tw/hid

require (
	github.com/asim/go-micro/plugins/broker/nats/v3 v3.0.0-20210416163442-a91d1f7a3dbb
	github.com/asim/go-micro/plugins/registry/nats/v3 v3.0.0-20210416163442-a91d1f7a3dbb
	github.com/asim/go-micro/plugins/transport/nats/v3 v3.0.0-20210416163442-a91d1f7a3dbb
	github.com/asim/go-micro/v3 v3.5.2
	github.com/dh1tw/remoteRotator v0.6.3-0.20210910212249-1d525322f67c
	github.com/dh1tw/remoteSwitch v0.2.2-0.20210910212220-2ebfcf967620
	github.com/dh1tw/streamdeck v0.1.4
	github.com/dh1tw/streamdeck-buttons v0.2.0
	github.com/nats-io/nats.go v1.12.1
)

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/dh1tw/hid v1.2.0 // indirect
	github.com/disintegration/gift v1.2.1 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/micro/cli/v2 v2.1.2 // indirect
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/image v0.0.0-20200618115811-c13761719519 // indirect
	golang.org/x/net v0.0.0-20210510120150-4163338589ed // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210502180810-71e4cd670f79 // indirect
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

replace github.com/gogo/protobuf v0.0.0-20190410021324-65acae22fc9 => github.com/gogo/protobuf v0.0.0-20190723190241-65acae22fc9d
