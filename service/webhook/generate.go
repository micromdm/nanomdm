package webhook

//go:generate go-jsonschema -p $GOPACKAGE --tags json --only-models --output event.go event.json
