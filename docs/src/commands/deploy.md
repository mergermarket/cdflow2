---
name: Deploy
menu: Commands
route: /commands/deploy
---

# Deploy

## Usage

```
cdflow2 [ GLOBALOPTS ] deploy [ OPTS ] ENV VERSION

Args:

  ENV                 - the environment being deployed to.
  VERSION             - the version being deployed (must match what was released).

Options:

  --plan-only | -p    - create the terraform plan only, don't apply.
  --new-state | -n    - allow run without a pre-existing tfstate file.
```

## Description

Terraform is configured as described in [common terraform setup](common-terraform-setup), followed by commands
equivalent to:

```shell
terraform plan \
    -input=false \
    -var-file release-metadata-VERSION.json \
    -var-file config/ENV.json \
    -out=plan-TIMESTAMP \
    infra/

terraform apply \
    -input=false \
    plan-TIMESTAMP
```

## Options

```
--plan-only : 
Perform the terraform plan command only. The terraform apply command is skipped.

--new-state :
Toggles a parameter sent to the config container. It is received before terraform commands are performed and used to trigger state existance validations.
```

"terraform",
		"plan",
		"-var-file=/build/release-metadata.json"

    "-var-file="+"config/common.json"
    "-var-file="+"config/" + args.EnvName + ".json"

    "-out="+"/build/" + util.RandomName("plan")
		"infra/",