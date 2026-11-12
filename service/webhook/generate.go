package webhook

// using https://github.com/jessepeterson/go-jsonschema/tree/micromdm
//go:generate go-jsonschema -p $GOPACKAGE --tags json --only-models --disable-omitzero --capitalization UUID,UDID,ID,IDs,URL --output event.go event.json
