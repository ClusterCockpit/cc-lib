package startup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	"github.com/ClusterCockpit/cc-lib/v2/ccTopology"
	"github.com/ClusterCockpit/cc-lib/v2/util"
	"github.com/nats-io/nats.go"
)

// EnvAuthToken overrides the configured HTTP endpoint auth token. It also
// accepts a "_FILE" variant naming a file that holds the token, so the token
// can come from a Docker or Kubernetes secret mount or from systemd
// LoadCredential; see util.SecretFromEnv for the precedence rules.
//
// The name is prefixed CC_ because cc-lib is linked into several applications,
// whose environments it must not silently claim names in.
const EnvAuthToken = "CC_STARTUP_AUTH_TOKEN"

// func StartupTopology(out chan lp.CCMessage) error {
// 	topo, err := ccTopology.LocalTopology()
// 	if err != nil {
// 		return fmt.Errorf("Failed to get local topology: %w", err)
// 	}

// 	topoJson, err := json.Marshal(topo)
// 	if err != nil {
// 		return fmt.Errorf("Failed to marshal topology: %w", err)
// 	}

// 	msg, err := lp.NewEvent("topology", map[string]string{
// 		"type": "node",
// 	}, nil, string(topoJson), time.Now())
// 	if err != nil {
// 		return fmt.Errorf("Failed to create event with topology: %w", err)
// 	}

// 	out <- msg
// 	return nil
// }

type CCStartupConfig struct {
	SendTopology bool `json:"send_topology,omitempty"`
	HttpEndpoint struct {
		URL       string `json:"url"`
		AuthToken string `json:"auth_token"`
	} `json:"http"`
	NatsEndpoint struct {
		URL      string `json:"url"`
		Subject  string `json:"subject"`
		NkeyFile string `json:"nkey_file"`
	} `json:"nats"`
}

func CCStartup(config json.RawMessage) error {
	conf := CCStartupConfig{
		SendTopology: true,
	}
	err := json.Unmarshal(config, &conf)
	if err != nil {
		err = fmt.Errorf("failed to read ccstartup configuration: %w", err)
		cclog.ComponentError("CCStartup", err.Error())
		return err
	}

	var out []byte
	if conf.SendTopology {
		topo, err := ccTopology.LocalTopology()
		if err != nil {
			err = fmt.Errorf("Failed to get local topology: %w", err)
			cclog.ComponentError("CCStartup", err.Error())
			return err
		}
		topoJson, err := json.Marshal(topo)
		if err != nil {
			err = fmt.Errorf("Failed to marshal topology: %w", err)
			cclog.ComponentError("CCStartup", err.Error())
			return err
		}
		out = topoJson
	}

	if len(out) > 0 {
		if len(conf.HttpEndpoint.URL) > 0 {
			// The token may come from the environment instead of the config
			// file. An unreadable secret file is fatal rather than a silent
			// fallback, so an unauthenticated request is never sent in its
			// place.
			authToken, err := util.SecretFromEnv(EnvAuthToken, conf.HttpEndpoint.AuthToken)
			if err != nil {
				err = fmt.Errorf("resolving %s: %w", EnvAuthToken, err)
				cclog.ComponentError("CCStartup", err.Error())
				return err
			}

			bodyReader := bytes.NewReader(out)
			req, err := http.NewRequest(http.MethodPost, conf.HttpEndpoint.URL, bodyReader)
			if err != nil {
				err = fmt.Errorf("failed to create HTTP request: %w", err)
				cclog.ComponentError("CCStartup", err.Error())
			} else {
				if len(authToken) > 0 {
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					err = fmt.Errorf("failed to send topology to %s: %w", conf.HttpEndpoint.URL, err)
					cclog.ComponentError("CCStartup", err.Error())
				} else {
					defer resp.Body.Close()
				}
			}
		}
		if len(conf.NatsEndpoint.URL) > 0 {
			var uinfo nats.Option = nil
			if len(conf.NatsEndpoint.NkeyFile) > 0 {
				uinfo = nats.UserCredentials(conf.NatsEndpoint.NkeyFile)
			}
			var client *nats.Conn
			if uinfo != nil {
				client, err = nats.Connect(conf.NatsEndpoint.URL, uinfo)
			} else {
				client, err = nats.Connect(conf.NatsEndpoint.URL)
			}
			if err != nil {
				err = fmt.Errorf("failed to connect to NATS URL %s: %w", conf.NatsEndpoint.URL, err)
				cclog.ComponentError("CCStartup", err.Error())
			} else {
				err = client.Publish(conf.NatsEndpoint.Subject, out)
				if err != nil {
					err = fmt.Errorf("failed to send topology to %s subject %s: %w", conf.NatsEndpoint.URL, conf.NatsEndpoint.Subject, err)
					cclog.ComponentError("CCStartup", err.Error())
				}
			}
		}
	}

	return nil
}
