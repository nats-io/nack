###################
### Variables
###################

variable REGISTRY {
  default = ""
}

# Comma delimited list of tags
variable TAGS {
  default = "latest"
}

variable CI {
  default = false
}

variable PUSH {
  default = false
}

###################
### Functions
###################

function "get_tags" {
  params = [image]
  result = [for tag in split(",", TAGS) : join("/", compact([REGISTRY, "${image}:${tag}"]))]
}

function "get_platforms_multiarch" {
  params = []
  result = (CI || PUSH) ? ["linux/amd64", "linux/arm/v6", "linux/arm/v7", "linux/arm64"] : []
}

function "get_output" {
  params = []
  result = (CI || PUSH) ? ["type=registry"] : ["type=docker"]
}

###################
### Groups
###################

group "default" {
  targets = [
    "jetstream-controller",
    "nats-boot-config",
    "nats-server-config-reloader"
  ]
}

###################
### Targets
###################

target "goreleaser" {
  contexts = {
    src = "."
  }
  args = {
    CI = CI
    PUSH = PUSH
    GITHUB_TOKEN = ""
  }
  dockerfile = "cicd/Dockerfile_goreleaser"
}

target "jetstream-controller" {
  contexts = {
    build   = "target:goreleaser"
    assets  = "cicd/assets"
  }
  args = {
    GO_APP = "jetstream-controller"
  }
  dockerfile  = "cicd/Dockerfile"
  platforms   = get_platforms_multiarch()
  tags        = get_tags("jetstream-controller")
  output      = get_output()
}

target "nats-boot-config-base" {
  contexts = {
    build   = "target:goreleaser"
    assets  = "cicd/assets"
  }
  args = {
    GO_APP = "nats-boot-config"
  }
  dockerfile  = "cicd/Dockerfile"
  platforms   = get_platforms_multiarch()
}

target "nats-boot-config" {
  inherits = ["nats-boot-config-base"]
  contexts = {
    base     = "target:nats-boot-config-base"
  }

  dockerfile-inline = <<EOT
ARG GO_APP
FROM base
RUN ln -s /usr/local/bin/$GO_APP /usr/local/bin/nats-pod-bootconfig
EOT

  platforms   = get_platforms_multiarch()
  tags        = get_tags("nats-boot-config")
  output      = get_output()
}

target "nats-server-config-reloader" {
  contexts = {
    build   = "target:goreleaser"
    assets  = "cicd/assets"
  }
  args = {
    GO_APP = "nats-server-config-reloader"
  }
  dockerfile  = "cicd/Dockerfile"
  platforms   = get_platforms_multiarch()
  tags        = get_tags("nats-server-config-reloader")
  output      = get_output()
}
