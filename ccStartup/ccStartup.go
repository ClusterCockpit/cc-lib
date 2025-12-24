package startup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	"github.com/ClusterCockpit/cc-lib/v2/ccTopology"
	"github.com/nats-io/nats.go"
)

// func StartupTopology(out chan lp.CCMessage) error {
// 	topo, err := ccTopology.LocalTopology()
// 	if err != nil {
// 		return fmt.Errorf("Failed to get local topology: %v", err.Error())
// 	}

// 	topoJson, err := json.Marshal(topo)
// 	if err != nil {
// 		return fmt.Errorf("Failed to marshal topology: %v", err.Error())
// 	}

// 	msg, err := lp.NewEvent("topology", map[string]string{
// 		"type": "node",
// 	}, nil, string(topoJson), time.Now())
// 	if err != nil {
// 		return fmt.Errorf("Failed to create event with topology: %v", err.Error())
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
		err = fmt.Errorf("failed to read ccstartup configuration: %s", err.Error())
		cclog.ComponentError("CCStartup", err.Error())
		return err
	}

	var out []byte
	if conf.SendTopology {
		topo, err := ccTopology.LocalTopology()
		if err != nil {
			err = fmt.Errorf("Failed to get local topology: %v", err.Error())
			cclog.ComponentError("CCStartup", err.Error())
			return err
		}
		topoJson, err := json.Marshal(topo)
		if err != nil {
			err = fmt.Errorf("Failed to marshal topology: %v", err.Error())
			cclog.ComponentError("CCStartup", err.Error())
			return err
		}
		out = topoJson
	}

	if len(out) > 0 {
		if len(conf.HttpEndpoint.URL) > 0 {
			bodyReader := bytes.NewReader(out)
			req, err := http.NewRequest(http.MethodPost, conf.HttpEndpoint.URL, bodyReader)
			if err != nil {
				err = fmt.Errorf("failed to create HTTP request: %s", err.Error())
				cclog.ComponentError("CCStartup", err.Error())
			} else {
				if len(conf.HttpEndpoint.AuthToken) > 0 {
					req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", conf.HttpEndpoint.AuthToken))
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					err = fmt.Errorf("failed to send topology to %s: %s", conf.HttpEndpoint.URL, err.Error())
					cclog.ComponentError("CCStartup", err.Error())
				}
				defer resp.Body.Close()
			}
		}
		if len(conf.NatsEndpoint.URL) > 0 {
			var client *nats.Conn = nil
			var uinfo nats.Option = nil
			if len(conf.NatsEndpoint.NkeyFile) > 0 {
				uinfo = nats.UserCredentials(conf.NatsEndpoint.NkeyFile)
			}
			if uinfo != nil {
				client, err = nats.Connect(conf.NatsEndpoint.URL, uinfo)
			} else {
				client, err = nats.Connect(conf.NatsEndpoint.URL)
			}
			if err != nil {
				err = fmt.Errorf("failed to connect to NATS URL %s: %s", conf.NatsEndpoint.URL, err.Error())
				cclog.ComponentError("CCStartup", err.Error())
			} else {
				err = client.Publish(conf.NatsEndpoint.Subject, out)
				if err != nil {
					err = fmt.Errorf("failed to send topology to %s subject %s: %s", conf.NatsEndpoint.URL, conf.NatsEndpoint.Subject, err.Error())
					cclog.ComponentError("CCStartup", err.Error())
				}
			}
		}
	}

	return nil
}
