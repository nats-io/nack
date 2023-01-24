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

variable image_base {
  default = "docker-image://alpine:3.17.1"
}

variable image_goreleaser {
  default = "docker-image://goreleaser/goreleaser:v1.14.1"
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
  result = CI ? ["linux/amd64", "linux/arm/v6", "linux/arm/v7", "linux/arm64"] : []
}

function "get_output" {
  params = []
  result = CI ? ["type=registry"] : ["type=docker"]
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
    goreleaser = image_goreleaser
    src = "."
  }
  args = {
    CI = CI
    GITHUB_TOKEN = ""
  }
  dockerfile = "cicd/Dockerfile_goreleaser"
}

target "jetstream-controller" {
  contexts = {
    base = image_base
    build = "target:goreleaser"
  }
  args = {
    GO_APP = "jetstream-controller"
  }
  dockerfile = "cicd/Dockerfile"
  platforms = get_platforms_multiarch()
  tags = get_tags("jetstream-controller")
  output = get_output()
}

target "nats-boot-config" {
  contexts = {
    base = image_base
    build = "target:goreleaser"
  }
  args = {
    GO_APP = "nats-boot-config"
  }
  dockerfile = "cicd/Dockerfile"
  platforms = get_platforms_multiarch()
  tags = get_tags("nats-boot-config")
  output = get_output()
}

target "nats-server-config-reloader" {
  contexts = {
    base = image_base
    build = "target:goreleaser"
  }
  args = {
    GO_APP = "nats-server-config-reloader"
  }
  dockerfile = "cicd/Dockerfile"
  platforms = get_platforms_multiarch()
  tags = get_tags("nats-server-config-reloader")
  output = get_output()
}
