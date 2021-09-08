package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

func main() {
	expressions := map[string]string{
		"default-gw":        "routes.running.destination==\"0.0.0.0/0\"",
		"base-iface-routes": "routes.running.next-hop-interface==matchers.default-gw.routes.running.0.next-hop-interface",
		"primary-nic":       "interfaces.name==matchers.default-gw.routes.running.0.next-hop-interface",
		//"bridge-routes":     "matchers.base-iface-routes | routes.running.next-hop-interface=\"br1\"",
		//"delete-primary-nic-routes": "matchers.base-iface-routes | routes.running.absent=true",
		//"composed-routes":           "matchers.delete-primary-nic-routes.routes.running + replacers.bridge-routes.routes.running",
	}
	asts := map[string]*Node{}
	for name, expression := range expressions {
		fmt.Println(expression)
		parser := NewParser(strings.NewReader(expression))
		ast, err := parser.Parse()
		if err != nil {
			panic(err)
		}
		asts[name] = ast
	}

	astsJSON, err := json.MarshalIndent(&asts, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", astsJSON)

	currentStateYAML := `
routes:
  running:
  - destination: 0.0.0.0/0
    next-hop-address: 192.168.100.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 1.1.1.0/24
    next-hop-address: 192.168.100.1
    next-hop-interface: eth1
    table-id: 254

interfaces:
  - name: eth1
    type: ethernet
    state: up
    ipv4:
      address:
      - ip: 10.244.0.1
        prefix-length: 24
      dhcp: false
      enabled: true
`

	currentState := map[interface{}]interface{}{}
	err = yaml.Unmarshal([]byte(currentStateYAML), &currentState)
	if err != nil {
		panic(err)
	}

	matchers, err := MatchersEmitter{AST: asts, currentState: currentState}.Emit()
	if err != nil {
		panic(err)
	}

	fmt.Println(matchers)
}
