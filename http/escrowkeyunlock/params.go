package escrowkeyunlock

import "net/url"

type EscrowKeyUnlockParams struct {
	// The deviceʼs serial number (required).
	// Provided in URL request string as "serial".
	Serial string

	// The device’s IMEI (omit for non-cellular devices).
	// Provided in URL request string as "imei".
	IMEI string

	// The device’s secondary IMEI (omit for non-cellular and
	// single-SIM devices).
	// Provided in URL request string as "imei2".
	IMEI2 string

	// The device’s MEID (omit for non-cellular devices).
	// Provided in URL request string as "meid".
	MEID string

	// Example: iPad4,1 (required).
	// Provided in URL request string as "productType".
	ProductType string

	// The client-supplied value for auditing purposes: a string that
	// identifies the name of the organization.
	// Provided in request form (body) as "orgName".
	OrgName string

	// The client-supplied value for auditing purposes: a string that
	// identifies the user requesting the removal (such as email, LDAP
	// ID, or name).
	// Provided in request form (body) as "guid".
	GUID string

	// The device’s bypass code.
	// Provided in request form (body) as "escrowKey".
	EscrowKey string
}

// Valid tests e for validity and required non-empty fields.
func (e *EscrowKeyUnlockParams) Valid() bool {
	if e == nil {
		return false
	}
	if e.Serial == "" || e.ProductType == "" || e.OrgName == "" || e.GUID == "" || e.EscrowKey == "" {
		return false
	}
	return true
}

// QueryParams builds query parameters for the "escrow key unlock" endpoint.
func (e *EscrowKeyUnlockParams) QueryParams() url.Values {
	q := make(url.Values)
	q.Set("serial", e.Serial)
	q.Set("productType", e.ProductType)
	if e.IMEI != "" {
		q.Set("imei", e.IMEI)
	}
	if e.IMEI2 != "" {
		q.Set("imei", e.IMEI2)
	}
	if e.MEID != "" {
		q.Set("imei", e.MEID)
	}
	return q
}

// QueryParams builds form parameters for the "escrow key unlock" endpoint.
func (e *EscrowKeyUnlockParams) FormParams() url.Values {
	f := make(url.Values)
	f.Set("orgName", e.OrgName)
	f.Set("guid", e.GUID)
	f.Set("escrowKey", e.EscrowKey)
	return f
}
