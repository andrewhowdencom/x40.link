# Requirements

## Operating Systems

The code is developed primarily on Linux-based operating systems (e.g., Debian)

## Languages

The following are programmed in:

| Language   | Role                                                                                                   |
|:-----------|:-------------------------------------------------------------------------------------------------------|
| [Go]       | The logic of the application itself                                                                    |
| [HCL]      | Infrastructure as code definitions                                                                     |
| [Markdown] | Documentation for the project                                                                          |
| [Python]   | Generate documentation (via mkdocs)                                                                    |
| [YAML]     | Configuration                                                                                          |

[Go]: https://go.dev/
[HCL]: https://github.com/hashicorp/hcl
[Markdown]: https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax
[Python]: https://www.python.org/
[YAML]: https://yaml.org/

## Tools

The following are the tools used to build, deploy, maintain, code, etc.

| Tool           | Purpose                                                                                           |
|:---------------|:--------------------------------------------------------------------------------------------------|
| [buf]          | Manage protocol buffers                                                                           |
| [cfssl]        | Generate PKI material                                                                             |
| [grpc-gateway] | Generate a HTTP+JSON gateway to the gRPC server                                                   |
| [hadolint]     | Lint the Dockerfile                                                                               |
| [podman]       | Build containers                                                                                  |
| [poetry]       | Manage dependencies & virtual environments in Python                                              |
| [protoc]       | Protocol Buffers                                                                                  |
| [task]         | Run defined tasks                                                                                 |
| [tar]          | Compress directories into archives                                                                |
| [tofu]         | Deploy infrastructure as code                                                                     |

[buf]: https://github.com/bufbuild/buf
[cfssl]: https://github.com/cloudflare/cfssl
[grpc-gateway]: https://github.com/grpc-ecosystem/grpc-gateway
[handolint]: https://github.com/hadolint/hadolint
[podman]: https://podman.io/
[poetry]: https://python-poetry.org/
[protoc]: https://grpc.io/docs/protoc-installation/
[task]: https://taskfile.dev/
[tar]: https://www.gnu.org/software/tar/
[tofu]: https://opentofu.org/