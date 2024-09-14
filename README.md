# TMC Go

`tmc-go` is a command line tool designed to streamline development environments and minimize production code.

## Installation
Install on your local machine like so:
```
go install github.com/IanRFerguson/tmc-go@v0.0.2
```

TMC Engineering has the `tmc-go Member Library .env` file in their 1Password vault. It should look like the `template.env` file in this repo.

## Usage

### Member Library CLI
You can check if a domain exists in our allowlist like so:
```
tmc-go member-library movementcooperative.org
```

Add a new record using the method flag:
```
tmc-go member-library mynewdomain.org --addDomain
```

## Dev Roadmap
- [x] - Member Library interface
- [ ] - 1Password interface for member service accounts
- [ ] - Automated `.env` methods that allow the user to define a profile
