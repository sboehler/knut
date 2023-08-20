package performance

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/commodity"
	"gopkg.in/yaml.v2"
)

type yamlUniverseFile map[string][]string

func LoadUniverseFromFile(reg *commodity.Registry, path string) (Universe, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadUniverse(reg, f)
}

func LoadUniverse(reg *commodity.Registry, r io.Reader) (Universe, error) {
	dec := yaml.NewDecoder(r)
	dec.SetStrict(true)
	var t yamlUniverseFile
	if err := dec.Decode(&t); err != nil {
		return nil, err
	}
	return fromYAML(reg, t)
}

type Universe map[*model.Commodity][]string

func fromYAML(reg *commodity.Registry, yaml yamlUniverseFile) (Universe, error) {
	universe := make(Universe)
	for class, commodities := range yaml {
		for _, name := range commodities {
			com, err := reg.Get(name)
			if err != nil {
				return nil, err
			}
			if _, ok := universe[com]; ok {
				return nil, fmt.Errorf("commodity %s already has a classification", com.Name())
			}
			universe[com] = append(strings.Split(class, ":"), com.Name())
		}
	}
	return universe, nil
}

func (un Universe) Locate(c *model.Commodity) []string {
	class, ok := un[c]
	if ok {
		return class
	}
	return []string{"Other", c.Name()}
}
