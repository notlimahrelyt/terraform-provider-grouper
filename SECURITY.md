# Security policy

## Reporting a vulnerability

Please report security problems privately, not in a public issue. Use
[**Report a vulnerability**](https://github.com/notlimahrelyt/terraform-provider-grouper/security/advisories/new)
on the repository's Security tab.

Include the provider version, a description of the problem, and steps to
reproduce it. Leave out real credentials, hostnames, and subject data. You will
get an acknowledgement within a few days, and a fix or mitigation plan once the
report is confirmed.

## Supported versions

Security fixes are made on the latest minor release. Upgrade to it to receive
them.

## Scope

In scope: anything in this repository that could expose credentials, write
secrets to state or logs, weaken TLS verification, or change memberships the
provider does not manage.

Out of scope: vulnerabilities in Grouper itself (report those to
[Internet2](https://spaces.at.internet2.edu/display/Grouper)) and in Terraform.
