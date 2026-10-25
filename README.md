# TMC Go

`tmc` is a command line tool designed to streamline development environments and minimize production code.

## Installation
Install on your local machine like so:
```
go install github.com/IanRFerguson/tmc-go/cmd/tmc@latest
```

TMC Engineering has the `tmc-go Member Library .env` file in their 1Password vault. It should look like the `template.env` file in this repo.

## Usage

### Member Library CLI
You can check if a domain exists in our allowlist like so:
```
tmc member-library movementcooperative.org
```

Add a new record using the method flag:
```
tmc member-library mynewdomain.org --addDomain
```

### Asana CLI
Store your [personal access token](https://app.asana.com/0/my-apps) and a default project (saved to `~/.tmc/asana/profile.json`):
```
tmc asana setup
```

Create a task in your default project, or another with `--project`:
```
tmc asana create "Fix the sync for ACME"
tmc asana create "Fix the sync for ACME" -d "Mobilize creds expired" --due tomorrow --assign-me
tmc asana create "Quarterly review" -p "Ops" --field "Priority=High"
```

If the project has custom fields you'll be prompted for them (leave blank to skip, or pass `--no-fields`). Setting `ASANA_API_KEY` in your environment overrides the stored key.

## Dev Roadmap
- [x] - Member Library interface
- [ ] - 1Password interface for member service accounts
- [ ] - Automated `.env` methods that allow the user to define a profile
