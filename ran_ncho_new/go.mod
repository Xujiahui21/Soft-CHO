module ran

go 1.21.3

require (
	git.cs.nctu.edu.tw/calee/sctp v1.1.0
	github.com/free5gc/nas v1.1.1
	github.com/free5gc/ngap v1.0.6
	github.com/free5gc/openapi v1.0.6
	github.com/free5gc/util v1.0.5-0.20230511064842-2e120956883b
	golang.org/x/net v0.17.0
	test v0.0.0-00010101000000-000000000000
)

require (
	github.com/aead/cmac v0.0.0-20160719120800-7af84192f0b1 // indirect
	github.com/antonfisher/nested-logrus-formatter v1.3.1 // indirect
	github.com/calee0219/fatal v0.0.1 // indirect
	github.com/evanphx/json-patch v0.5.2 // indirect
	github.com/free5gc/aper v1.0.4 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/golang-jwt/jwt v3.2.1+incompatible // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.mongodb.org/mongo-driver v1.8.4 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
)

replace (
	github.com/free5gc/amf => ../free5gc/NFs/amf
	github.com/free5gc/ausf => ../free5gc/NFs/ausf
	github.com/free5gc/n3iwf => ../free5gc/NFs/n3iwf
	github.com/free5gc/nrf => ../free5gc/NFs/nrf
	github.com/free5gc/nssf => ../free5gc/NFs/nssf
	github.com/free5gc/pcf => ../free5gc/NFs/pcf
	github.com/free5gc/smf => ../free5gc/NFs/smf
	github.com/free5gc/udm => ../free5gc/NFs/udm
	github.com/free5gc/udr => ../free5gc/NFs/udr
	test => ../free5gc/test
)
