package main

import (
	"io/ioutil"
	"encoding/json"
	"strconv"
)

type ConfigMap map[string]interface{}

var Config ConfigMap

func LoadConfig() error {
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		return err
	}
	Config = make(map[string]interface{})
	err = json.Unmarshal(data, &Config)
	return err
}

// dv means default value
// this function will auto convert number
func (c ConfigMap) getInt(k string, dv int) int {
	v, ok := (map[string]interface{})(c)[k]
	if !ok { return dv }
	switch v.(type) {
	case float64:
		return int(v.(float64))
	case string:
		i, err := strconv.Atoi(v.(string))
		if err != nil { return dv }
		return i
	default:
		return dv
	}
}

// dv means default value
// this function will auto convert number
func (c ConfigMap) getFloat(k string, dv float64) float64 {
	v, ok := (map[string]interface{})(c)[k]
	if !ok { return dv }
	switch v.(type) {
	case float64:
		return v.(float64)
	case string:
		d, err := strconv.ParseFloat(v.(string), 64)
		if err != nil { return dv }
		return d
	default:
		return dv
	}
}

// dv means default value
// this function will auto convert number to string
func (c ConfigMap) getString(k string, dv string) string {
	v, ok := (map[string]interface{})(c)[k]
	if !ok { return dv }
	switch v.(type) {
	case float64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	case string:
		return v.(string)
	default:
		return dv
	}
}

func (c ConfigMap) getConfigMap(k string) ConfigMap {
	v, ok := (map[string]interface{})(c)[k]
	if !ok { return nil }
	cm, ok := v.(map[string]interface{})
	return ConfigMap(cm)
}