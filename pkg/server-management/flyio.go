package servermanagement

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var BASE_URL = "https://api.machines.dev"

func machineCreateUrl() string {
    return fmt.Sprintf("%s/v1/apps/vim-arcade/machines", BASE_URL)
}

func machineStart(machineId string) string {
    return fmt.Sprintf("%s/start", machine(machineId))
}

func machineStop(machineId string) string {
    return fmt.Sprintf("%s/stop", machine(machineId))
}

func machineDestroy(machineId string) string {
    return fmt.Sprintf("%s?force=true", machine(machineId))
}

func machine(machineId string) string {
    return fmt.Sprintf("%s/v1/apps/vim-arcade/machines/%s", BASE_URL, machineId)
}

//{"id":"1852414f4125d8","name":"vim-arcade","state":"created","region":"den","instance_id":"01J6ZE56EA12S649SSKRK71PF6","private_ip":"fdaa:3:c60a:a7b:5:5607:a0b9:2","config":{"env":{"APP_ENV":"production"},"init":{},"guest":{"cpu_kind":"shared","cpus":1,"memory_mb":256},"services":[{"protocol":"tcp","internal_port":8080,"ports":[{"port":80,"handlers":["http"]}],"force_instance_key":null}],"image":"registry.fly.io/vim-arcade:deployment-01J6XBCR5F95VZAH6REWNP2XBC"},"incomplete_config":null,"image_ref":{"registry":"registry.fly.io","repository":"vim-arcade","tag":"deployment-01J6XBCR5F95VZAH6REWNP2XBC","digest":"sha256:ac31956327e300c624741d92b6537f766890d239f6ef8e44fc20edd1c672f94f","labels":null},"created_at":"2024-09-04T21:13:27Z","updated_at":"2024-09-04T21:13:27Z","events":[{"id":"01J6ZE56FARDMV304HFVVAT8PK","type":"launch","status":"created","source":"user","timestamp":1725484407274}],"host_status":"ok"}
type MachineCreateResponse struct {
    Id string `json:"id"`
    InstanceID string `json:"instance_id"`
}

func (m *MachineCreateResponse) String() string {
    return fmt.Sprintf("Id: %s -- InstanceID: %s", m.Id, m.InstanceID)
}

func createMachine() (*MachineCreateResponse, error) {
	body := []byte(`{
      "config": {
        "image": "registry.fly.io/vim-arcade:deployment-01J6XBCR5F95VZAH6REWNP2XBC",
        "env": {
          "APP_ENV": "production"
        },
        "services": [
          {
            "ports": [
                {
                "port": 443,
                "handlers": [
                  "tls",
                  "http"
                ]
              },
              {
                "port": 80,
                "handlers": [
                  "http"
                ]
              }
            ],
            "protocol": "tcp",
            "internal_port": 8080
          }
        ]
      }
	}`)

	r, err := http.NewRequest("POST", machineCreateUrl(), bytes.NewBuffer(body))
	if err != nil {
        return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("FLY_IO_ORG_TOKEN")))

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
        return nil, err
	}

	defer res.Body.Close()

    machineResponse := MachineCreateResponse{}
    b, err := io.ReadAll(res.Body)
    if err != nil {
        return nil, err
    }
    fmt.Printf("Machine: %s -- %d\n", string(b), res.StatusCode)
    err = json.Unmarshal(b, &machineResponse)
    if err != nil {
        return nil, err
    }

    return &machineResponse, nil
}

func getMachine(machineId string) (string, error) {
	r, err := http.NewRequest("GET", machine(machineId), bytes.NewBuffer([]byte{}))
	if err != nil {
        return "", err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("FLY_IO_ORG_TOKEN")))

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
        return "", err
	}

	defer res.Body.Close()

    b, err := io.ReadAll(res.Body)
    if err != nil {
        return "", err;
    }

    return string(b), err
}

func stopMachine(machineId string) (string, error) {
	r, err := http.NewRequest("POST", machineStop(machineId), bytes.NewBuffer([]byte{}))
	if err != nil {
        return "", err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("FLY_IO_ORG_TOKEN")))

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
        return "", err
	}

	defer res.Body.Close()

    b, err := io.ReadAll(res.Body)
    if err != nil {
        return "", err;
    }

    return string(b), err
}

func startMachine(machineId string) (string, error) {
	r, err := http.NewRequest("POST", machineStart(machineId), bytes.NewBuffer([]byte{}))
	if err != nil {
        return "", err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("FLY_IO_ORG_TOKEN")))

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
        return "", err
	}

	defer res.Body.Close()

    b, err := io.ReadAll(res.Body)
    if err != nil {
        return "", err;
    }

    return string(b), err
}

func destroyMachine(machineId string) (string, error) {
	r, err := http.NewRequest("DELETE", machineDestroy(machineId), bytes.NewBuffer([]byte{}))
	if err != nil {
        return "", err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("FLY_IO_ORG_TOKEN")))

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
        return "", err
	}

	defer res.Body.Close()

    b, err := io.ReadAll(res.Body)
    if err != nil {
        return "", err;
    }

    return string(b), err
}

