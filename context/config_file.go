package context

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"bytes"
	"gopkg.in/yaml.v3"
	"os/exec"
)

const defaultHostname = "github.com"

type configEntry struct {
	User  string
	Token string `yaml:"oauth_token"`
}
type execEntry struct {
	Exec struct {
		Command string   `yaml:"command"`
		Args    []string `yaml:"args"`
	} `yaml:"exec"`
}

func parseOrSetupConfigFile(fn string) (*configEntry, error) {
	entry, err := parseConfigFile(fn)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return setupConfigFile(fn)
	}
	return entry, err
}

func parseConfigFile(fn string) (*configEntry, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseConfig(f)
}

// ParseDefaultConfig reads the configuration file
func ParseDefaultConfig() (*configEntry, error) {
	return parseConfigFile(configFile())
}

func parseConfig(r io.Reader) (*configEntry, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var config yaml.Node
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	if len(config.Content) < 1 {
		return nil, fmt.Errorf("malformed config")
	}
	for i := 0; i < len(config.Content[0].Content)-1; i = i + 2 {
		if config.Content[0].Content[i].Value == defaultHostname {

			var execEntries []execEntry
			var entries []configEntry
			err := config.Content[0].Content[i+1].Decode(&execEntries)
			// If content is execOptions, run the command to get the auth config
			if err == nil {
				entry := execEntries[0]
				stdout := &bytes.Buffer{}
				stderr := &bytes.Buffer{}
				cmd := exec.Command(entry.Exec.Command, entry.Exec.Args...)
				cmd.Stdout = stdout
				cmd.Stderr = stderr
				if err := cmd.Run(); err != nil {
					return nil, fmt.Errorf("exec: %v with stdout %q and stderr %q", err, stdout.Bytes(), stderr.Bytes())
				}

				var configEntry configEntry
				err = yaml.Unmarshal(stdout.Bytes(), &configEntry)
				if err != nil {
					return nil, err
				}
				entries = append(entries, configEntry)
			} else {
				err = config.Content[0].Content[i+1].Decode(&entries)
			}

			if err != nil {
				return nil, err
			}

			return &entries[0], nil
		}
	}
	return nil, fmt.Errorf("could not find config entry for %q", defaultHostname)
}
