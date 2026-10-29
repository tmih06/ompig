# Containerization

Ompig runs with the permissions of its process. Use a container or virtual machine when you need an operating-system security boundary.

A container can limit filesystem, process, device, and network access. It does not make model output or extension source trustworthy.

## Build a Ompig image

Ompig does not publish a container image. Build one from the source tree with a multi-stage Dockerfile:

```dockerfile
FROM golang:1.27.3 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -o /out/ompig ./cmd/ompig

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends bash ca-certificates git ripgrep \
  && rm -rf /var/lib/apt/lists/*
RUN useradd --create-home --uid 10001 ompig
COPY --from=build /out/ompig /usr/local/bin/ompig
USER ompig
WORKDIR /workspace
ENTRYPOINT ["ompig"]
```

`ca-certificates` lets Ompig verify the TLS certificates of model providers. `ripgrep` serves the `grep` tool. Without it, and without `fd` for the `find` tool, Ompig downloads the missing tool on first use, as Pi does, which fails without network access.

Pin base images by digest for a release build. Generate an SBOM and scan the final image before publication.

## Preserve state deliberately

Ompig stores user state below:

```text
~/.ompig
```

Mount a named volume when Sessions and settings must survive container replacement:

```bash
docker run --rm -it \
  -e ANTHROPIC_API_KEY \
  -v ompig-home:/home/ompig/.ompig \
  -v "$PWD:/workspace" \
  ompig-local
```

Replace `ANTHROPIC_API_KEY` with the credential your provider requires. Environment variables passed into a container are available to every process inside it.

Mounting your host `~/.ompig` gives the container access to host credentials, Sessions, settings, Packages, and extension state. Use a separate volume when that access is not required.

## Workspace access

Mount only the project paths that Ompig needs. Use a read-only mount when the agent should inspect but not modify files:

```bash
docker run --rm -it \
  -v "$PWD:/workspace:ro" \
  ompig-local -p "Review this repository"
```

A read-only workspace does not prevent access to another writable mount.

## Network access

Disable network access when the task does not need model or external service calls:

```bash
docker run --rm -it --network none ompig-local
```

Normal model inference requires network access unless the model endpoint is available through an explicitly permitted local network.

Use network policy or a proxy when the container must reach only approved endpoints.

## Extensions in containers

Source extensions can require a compiler or interpreter. Include only the required runtime in the image, or use a prebuilt Ompiglet Binary.

A Ompig Standard Binary compiles all selected extensions into the executable. It still needs external model services, secrets, and configuration that the Ompiglet record declares.

Process isolation inside a container is not the same as isolation from the container. A subprocess extension normally has the same container permissions as Ompig.

## Ompiglet agent environments

A Ompiglet can declare an `agentEnv` requirement for the whole application. This requirement is separate from the code-execution sandbox and from extension process placement.

Ompig must enter or verify the required environment before startup. It must not silently use the host when the requirement cannot be met.

See [Ompiglets](/docs/latest/ompiglets#required-agent-environment).

## Secrets

Inject secrets at runtime. Do not bake them into an image or copy them into a build layer.

Use:

- environment variables;
- owner-only mounted files;
- an approved external secret resolver.

Review image history and generated SBOMs before release.

## Rootless execution

Run as a non-root user unless the task requires a specific privileged operation. Do not grant broad host mounts, Docker socket access, device access, or additional Linux capabilities by default.

## Related documentation

- [Security](/docs/latest/security)
- [Ompiglets](/docs/latest/ompiglets)
- [Ompiglet Binaries](/docs/latest/ompiglet-binaries)
- [Extensions](/docs/latest/extensions)
- [upstream Pi's isolation documentation](https://pi.dev/docs/latest/containerization), which also covers Docker Sandboxes, OpenShell and the Gondolin extension for Pi
