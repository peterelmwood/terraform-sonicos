# terraform-provider-sonicos

A [Terraform](https://www.terraform.io) provider for managing **SonicOS 7+**
firewall configuration declaratively, through the appliance's built-in REST
API.

SonicWall does not publish an official Terraform provider. SonicOS 7 is,
however, entirely API-driven — the web UI itself drives the same REST API — so
the platform is a good fit for declarative, plan/apply, GitOps-style management.
This provider is built with the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
and maps Terraform resources one-to-one onto SonicOS API objects.

> Status: early, focused on the highest-volume object types. See
> [Supported resources](#supported-resources) and [Limitations](#limitations).

## Why these design choices

Two SonicOS behaviors shape the provider:

- **Pending / commit model.** Writes (POST/PUT/DELETE) do not take effect
  immediately; they accumulate in a *pending* configuration that is activated
  with a single commit call (`POST /api/sonicos/config/pending`). Terraform's
  may execute resource operations concurrently (subject to the dependency graph
  and `-parallelism`) and exposes no global end-of-apply hook, so the provider
  commits after each resource write by default
  (`commit_on_apply = true`). Set `commit_on_apply = false` to stage changes
  without committing and drive the commit yourself when you want the whole
  apply to land as one transaction.
- **Single API session, full-admin only.** SonicOS permits one API session at a
  time and requires a full-admin identity. Use a dedicated automation account
  and serialize Terraform runs (avoid `-parallelism` against the same
  appliance, and don't run two pipelines concurrently).

## Supported resources

| Type | Terraform | SonicOS endpoint |
| --- | --- | --- |
| Resource | `sonicos_address_object` | `/address-objects/ipv4` |
| Resource | `sonicos_service_object` | `/service-objects` |
| Resource | `sonicos_zone` | `/zones` |
| Resource | `sonicos_access_rule` | `/access-rules/ipv4` |
| Data source | `sonicos_address_object` | `/address-objects/ipv4` |

These cover most day-to-day change volume. The architecture (a typed client in
[`internal/client`](internal/client) plus one file per resource in
[`internal/provider`](internal/provider)) is designed so additional object
types — NAT policies, interfaces, IPv6 variants — follow the same pattern.

## Provider configuration

```hcl
terraform {
  required_providers {
    sonicos = {
      source = "peterelmwood/sonicos"
    }
  }
}

provider "sonicos" {
  host     = "https://192.0.2.1:8443" # or SONICOS_HOST
  username = "tf-automation"          # or SONICOS_USERNAME
  password = var.sonicos_password     # or SONICOS_PASSWORD
  insecure = true                     # self-signed appliance cert
}
```

| Attribute | Env var | Default | Description |
| --- | --- | --- | --- |
| `host` | `SONICOS_HOST` | — | Appliance management URL or host. |
| `username` | `SONICOS_USERNAME` | — | Full-admin API username. |
| `password` | `SONICOS_PASSWORD` | — | API password (sensitive). |
| `insecure` | `SONICOS_INSECURE` | `false` | Skip TLS verification. |
| `commit_on_apply` | — | `true` | Commit the pending config after each write. |

See [`examples/`](examples) for full resource examples.

## Building and local development

Requires Go 1.23+.

```sh
make build   # build ./terraform-provider-sonicos
make test    # run unit tests
make vet     # go vet
make install # build and copy into your GOBIN for dev_overrides
```

To use a locally built provider, add a `dev_overrides` block to your Terraform
CLI config (`~/.terraformrc`) pointing at your `GOBIN`:

```hcl
provider_installation {
  dev_overrides {
    "peterelmwood/sonicos" = "/home/you/go/bin"
  }
  direct {}
}
```

Then run `terraform plan` / `apply` against a lab appliance — `terraform init`
is skipped under dev overrides.

## How it works

- [`internal/client`](internal/client) — a small typed HTTP client for the
  SonicOS REST API: Basic-auth login with session cookie, the named-collection
  JSON envelope each object uses, per-object CRUD, `404`-as-drift detection, and
  `CommitPending`.
- [`internal/provider`](internal/provider) — the Terraform provider: `Configure`
  authenticates and shares the client; each resource implements Create / Read /
  Update / Delete / ImportState, with `Read` faithfully reflecting the appliance
  so Terraform's plan-diff and drift detection work for free.
- [`internal/lint`](internal/lint) + [`cmd/tfsonicos`](cmd/tfsonicos) — the
  configuration linter (see below).

## Config linting

Some mistakes only become visible when you look at the *whole* configuration —
a rule pointing at an object defined in another file, a rule that can never
match because a broader one precedes it, or an accidental any→any allow.
Terraform's per-resource validation can't see across resources, so the linter
fills that gap. It works at two levels:

**In `terraform plan` (single-resource).** The provider validates the format of
IP-bearing attributes during planning, so a malformed host IP or a
non-contiguous subnet mask is reported as a clear diagnostic instead of an
opaque appliance rejection at apply time.

**Whole-config (`tfsonicos lint`).** A standalone CLI reads the JSON from
`terraform show -json` (of a plan file or current state) and applies
cross-resource rules:

```sh
make build-lint                                   # build ./tfsonicos
terraform plan -out plan.tfplan
terraform show -json plan.tfplan | ./tfsonicos lint
# or lint current state:
terraform show -json | ./tfsonicos lint
# machine-readable, and fail CI on warnings too:
terraform show -json plan.tfplan | ./tfsonicos lint -format json -strict
```

It exits non-zero when any **error**-severity finding is present (add `-strict`
to also fail on warnings), so it can gate a pipeline. Rule categories:

| Category | Examples | Severity |
| --- | --- | --- |
| Referential integrity | a rule/NAT policy references an address, service, or zone not defined in the config | error (zones/interfaces that may pre-exist on the appliance: warning) |
| Field & format | invalid IPv4/IPv6, bad subnet mask, port out of range, reversed range | error |
| Duplicate names | two resources declare the same object/zone/rule name (the appliance requires uniqueness) | error |
| Shadowed rules | a rule can never match because an earlier, broader rule on the same zone pair handles its traffic | warning |
| Overly-permissive | any→any allow rules; allows inbound from an untrusted zone on any service | warning |

Built-in zones (`LAN`, `WAN`, `DMZ`, …) are treated as always present, so
referencing them is never flagged.

## Limitations

- IPv4 only for address objects and access rules so far; IPv6 endpoints exist on
  the appliance and can be added with the same pattern.
- The object models cover common fields, not every appliance attribute. They
  should be cross-checked against the Swagger/OpenAPI document on your specific
  firmware (navigate to **Home | API** on the appliance) before production use.
- Access rules are matched back to their appliance-assigned UUID by **name**
  immediately after creation, so rule names must be unique.
- No acceptance tests against a live appliance are included; the unit tests
  exercise the client against an in-process HTTP server. Validate against a lab
  device before managing production.

## License

MIT — see [LICENSE](LICENSE).
