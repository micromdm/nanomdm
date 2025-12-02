package api

//go:generate oa2js -o ErrorResponse.json ../../docs/openapi.yaml ErrorResponse
//go:generate oa2js -o PushCertResponse.json ../../docs/openapi.yaml PushCertResponse
//go:generate go-jsonschema -p $GOPACKAGE --tags json --only-models --output schema.go ErrorResponse.json PushCertResponse.json
//go:generate rm -f ErrorResponse.json PushCertResponse.json
