package mdm

import "testing"

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error")
	}
}

func assertNilError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

const (
	certQueryEscaped = "-----BEGIN+CERTIFICATE-----%0AMIIC1TCCAb2gAwIBAgIJAOOl7VQeisl5MA0GCSqGSIb3DQEBCwUAMBoxGDAWBgNV%0ABAMMD21kbS5leGFtcGxlLm9yZzAeFw0yNTAxMzAxOTA3NDhaFw0yNjAxMzAxOTA3%0ANDhaMBoxGDAWBgNVBAMMD21kbS5leGFtcGxlLm9yZzCCASIwDQYJKoZIhvcNAQEB%0ABQADggEPADCCAQoCggEBAMMuJRNUCmgdKs6W%2BdVna8ftPokGsm7xN7xGG%2BHcAs41%0AI2ImgcrbXG35%2Fb9OWlG3%2FFxAJuXwWaajcRVcfdXHeBwinsdiywzxWDjaL30tjCaA%0A4%2FgIHCamXEmpnxdC%2FG41GNSYMAjM6Qo1hUeLuvdKtGskTIsY0Bn12%2BX9VvgFK%2Fw5%0A5XCqdNXWZtNJm%2B6xnJn2lWo%2BMQ1pCGT9o2vkCt7IXz5VeCFFsRAFs58cUUIvH%2FNu%0A1VL2wOUON2qbms0VnLF0oLvFwZG1u25TSzMOMJTM2s0HjjnP5Ef%2Fmx4QvLEXYuwv%0AH04lK2LP3iQvO0dYRildZ3Te5fAcgHgqNeqk8S3gg3ECAwEAAaMeMBwwGgYDVR0R%0ABBMwEYIPbWRtLmV4YW1wbGUub3JnMA0GCSqGSIb3DQEBCwUAA4IBAQAVuu9eLtd6%0A09JBMHIcFUA1h0MvnPZ7bJQCYjIvh7CIwl7SBlFiaQ3gIahelAR5pqdOxpqoYZdj%0Agkns4qH4GH6NDORoVl7WPPIpT4s9cD%2BzaEzMrc1ZmzPwEksBl89yfkB5QH0kXhe4%0AjpSxtcYOwGQ7BOJDDqhqiI47NnTF5Xsy53OocauXVXSdDYfHNxAokijKMWEQRnGs%0A2Gjc5jF%2Fse%2FojXko3pCP71Q4lGFRo%2FyqGUmwZ8Ul%2F3Bm%2FH4nk%2FrvcYbcXToIpDuE%0A4ioXhsGZD%2FtfDKSGd4QyEL5sBb%2F8ULuC%2By1nolRY7zZTc3eUVEJUM7li4JHB2s5r%0AKGNh7rtCvJQw%0A-----END+CERTIFICATE-----%0A"
	certRFC9440      = ":MIIC1TCCAb2gAwIBAgIJAOOl7VQeisl5MA0GCSqGSIb3DQEBCwUAMBoxGDAWBgNVBAMMD21kbS5leGFtcGxlLm9yZzAeFw0yNTAxMzAxOTA3NDhaFw0yNjAxMzAxOTA3NDhaMBoxGDAWBgNVBAMMD21kbS5leGFtcGxlLm9yZzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMMuJRNUCmgdKs6W+dVna8ftPokGsm7xN7xGG+HcAs41I2ImgcrbXG35/b9OWlG3/FxAJuXwWaajcRVcfdXHeBwinsdiywzxWDjaL30tjCaA4/gIHCamXEmpnxdC/G41GNSYMAjM6Qo1hUeLuvdKtGskTIsY0Bn12+X9VvgFK/w55XCqdNXWZtNJm+6xnJn2lWo+MQ1pCGT9o2vkCt7IXz5VeCFFsRAFs58cUUIvH/Nu1VL2wOUON2qbms0VnLF0oLvFwZG1u25TSzMOMJTM2s0HjjnP5Ef/mx4QvLEXYuwvH04lK2LP3iQvO0dYRildZ3Te5fAcgHgqNeqk8S3gg3ECAwEAAaMeMBwwGgYDVR0RBBMwEYIPbWRtLmV4YW1wbGUub3JnMA0GCSqGSIb3DQEBCwUAA4IBAQAVuu9eLtd609JBMHIcFUA1h0MvnPZ7bJQCYjIvh7CIwl7SBlFiaQ3gIahelAR5pqdOxpqoYZdjgkns4qH4GH6NDORoVl7WPPIpT4s9cD+zaEzMrc1ZmzPwEksBl89yfkB5QH0kXhe4jpSxtcYOwGQ7BOJDDqhqiI47NnTF5Xsy53OocauXVXSdDYfHNxAokijKMWEQRnGs2Gjc5jF/se/ojXko3pCP71Q4lGFRo/yqGUmwZ8Ul/3Bm/H4nk/rvcYbcXToIpDuE4ioXhsGZD/tfDKSGd4QyEL5sBb/8ULuC+y1nolRY7zZTc3eUVEJUM7li4JHB2s5rKGNh7rtCvJQw:"
)

func TestExtractRFC9440(t *testing.T) {
	_, err := ExtractRFC9440("")
	assertError(t, err)

	_, err = ExtractRFC9440(":")
	assertError(t, err)

	_, err = ExtractRFC9440(":INVALID:")
	assertError(t, err)

	_, err = ExtractRFC9440(certRFC9440)
	assertNilError(t, err)
}

func TestQueryEscapedPEM(t *testing.T) {
	_, err := ExtractQueryEscapedPEM("")
	assertError(t, err)

	_, err = ExtractQueryEscapedPEM("%GK") // invalid query escape code
	assertError(t, err)

	_, err = ExtractQueryEscapedPEM("INVALID")
	assertNilError(t, err)

	_, err = ExtractQueryEscapedPEM(certQueryEscaped)
	assertNilError(t, err)
}
