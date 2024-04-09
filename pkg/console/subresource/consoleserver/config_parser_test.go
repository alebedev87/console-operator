package consoleserver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/openshift/api/operator/v1"
)

func TestConfigParser(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		outputBuilder *ConsoleServerCLIConfigBuilder
	}{
		{
			name: "Parser should parse a nominal config",
			input: `apiVersion: console.openshift.io/v1
auth:
  clientID: console
  clientSecretFile: /var/oauth-config/clientSecret
  logoutRedirect: https://foobar.com/logout
clusterInfo:
  consoleBaseAddress: https://console-openshift-console.apps.foobar.com
  masterPublicURL: https://foobar.com/api
customization:
  branding: okd
  documentationBaseURL: https://foobar.com/docs
kind: ConsoleConfig
providers:
  statuspageID: status-12345
servingInfo:
  bindAddress: https://[::]:8443
  certFile: /var/serving-cert/tls.crt
  keyFile: /var/serving-cert/tls.key
session: {}
`,
			outputBuilder: (&ConsoleServerCLIConfigBuilder{}).
				Host("https://console-openshift-console.apps.foobar.com").
				LogoutURL("https://foobar.com/logout").
				Brand(v1.BrandOKD).
				DocURL("https://foobar.com/docs").
				APIServerURL("https://foobar.com/api").
				StatusPageID("status-12345"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := (&ConsoleYAMLParser{}).Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected unmarshaling error: %v", err)
			}
			output := tt.outputBuilder.Config()
			if diff := cmp.Diff(output, *input); len(diff) > 0 {
				t.Error(diff)
			}
		})
	}
}
