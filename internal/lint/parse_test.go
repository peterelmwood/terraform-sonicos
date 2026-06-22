package lint

import "testing"

// A representative slice of `terraform show -json` of a plan, including a child
// module and a non-SonicOS resource that must be ignored.
const sampleShowJSON = `{
  "format_version": "1.0",
  "planned_values": {
    "root_module": {
      "resources": [
        {
          "address": "sonicos_address_object.web",
          "type": "sonicos_address_object",
          "name": "web",
          "values": { "name": "web", "zone": "DMZ", "type": "host", "host": "10.1.1.5" }
        },
        {
          "address": "sonicos_access_rule.allow_web",
          "type": "sonicos_access_rule",
          "name": "allow_web",
          "values": {
            "name": "allow-web", "from": "WAN", "to": "DMZ", "action": "allow",
            "enable": true, "destination_name": "web", "service_name": "https"
          }
        },
        {
          "address": "random_pet.example",
          "type": "random_pet",
          "name": "example",
          "values": { "length": 2 }
        }
      ],
      "child_modules": [
        {
          "address": "module.svc",
          "resources": [
            {
              "address": "module.svc.sonicos_service_object.https",
              "type": "sonicos_service_object",
              "name": "https",
              "values": { "name": "https", "protocol": "tcp", "port_begin": 443, "port_end": 443 }
            }
          ]
        }
      ]
    }
  }
}`

func TestParseShowJSON(t *testing.T) {
	cfg, err := Parse([]byte(sampleShowJSON))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.AddressObjects) != 1 || cfg.AddressObjects[0].Host != "10.1.1.5" {
		t.Errorf("address objects: %+v", cfg.AddressObjects)
	}
	if len(cfg.AccessRules) != 1 || cfg.AccessRules[0].ServiceName != "https" {
		t.Errorf("access rules: %+v", cfg.AccessRules)
	}
	// Service object lives in a child module and must still be picked up.
	if len(cfg.ServiceObjects) != 1 || cfg.ServiceObjects[0].PortBegin != 443 {
		t.Errorf("service objects: %+v", cfg.ServiceObjects)
	}
	// The non-SonicOS resource must be ignored.
	if len(cfg.Interfaces)+len(cfg.NATPolicies)+len(cfg.Zones) != 0 {
		t.Errorf("unexpected extra resources parsed")
	}

	// End-to-end: this config is clean.
	if f := Lint(cfg); len(f) != 0 {
		t.Errorf("expected clean lint, got %v", f)
	}
}

func TestParseStateValuesFallback(t *testing.T) {
	// `terraform show -json` of state uses "values" rather than "planned_values".
	const stateJSON = `{
      "values": { "root_module": { "resources": [
        { "address": "sonicos_zone.dmz", "type": "sonicos_zone", "name": "dmz",
          "values": { "name": "DMZ", "security_type": "public" } }
      ] } }
    }`
	cfg, err := Parse([]byte(stateJSON))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Zones) != 1 || cfg.Zones[0].Name != "DMZ" {
		t.Errorf("zones: %+v", cfg.Zones)
	}
}

func TestParseEmptyInput(t *testing.T) {
	cfg, err := Parse([]byte(`{}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.AddressObjects) != 0 {
		t.Errorf("expected empty config")
	}
}
