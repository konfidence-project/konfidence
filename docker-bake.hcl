variable "TAG" {
  default = "dev"
}

variable "COMMIT_SHA" {
  default = ""
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["star-operator", "galaxy-operator"]
}

target "star-operator" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = concat(
    ["${REGISTRY}/star-operator:${TAG}"],
    COMMIT_SHA != "" ? ["${REGISTRY}/star-operator:${COMMIT_SHA}"] : [],
  )
  args       = { OPERATOR_NAME = "star" }
}

target "galaxy-operator" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = concat(
    ["${REGISTRY}/galaxy-operator:${TAG}"],
    COMMIT_SHA != "" ? ["${REGISTRY}/galaxy-operator:${COMMIT_SHA}"] : [],
  )
  args       = { OPERATOR_NAME = "galaxy" }
}
