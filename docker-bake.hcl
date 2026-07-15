variable "TAG" {
  default = "dev"
}

variable "COMMIT_SHA" {
  default = ""
}

variable "REGISTRY" {
  default = "ghcr.io"
}

group "default" {
  targets = ["star-operator", "galaxy-operator", "api", "konfidence-ui"]
}

target "star-operator" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = concat(
    ["${REGISTRY}/konfidence-project/star-operator:${TAG}"],
    COMMIT_SHA != "" ? ["${REGISTRY}/konfidence-project/star-operator:${COMMIT_SHA}"] : [],
  )
  args       = { OPERATOR_NAME = "star" }
}

target "galaxy-operator" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = concat(
    ["${REGISTRY}/konfidence-project/galaxy-operator:${TAG}"],
    COMMIT_SHA != "" ? ["${REGISTRY}/konfidence-project/galaxy-operator:${COMMIT_SHA}"] : [],
  )
  args       = { OPERATOR_NAME = "galaxy" }
}

target "api" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = concat(
    ["${REGISTRY}/konfidence-project/api:${TAG}"],
    COMMIT_SHA != "" ? ["${REGISTRY}/konfidence-project/api:${COMMIT_SHA}"] : [],
  )
  args       = { OPERATOR_NAME = "api" }
}

target "konfidence-ui" {
  context    = "."
  dockerfile = "Dockerfile.ui"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = concat(
    ["${REGISTRY}/konfidence-project/konfidence-ui:${TAG}"],
    COMMIT_SHA != "" ? ["${REGISTRY}/konfidence-project/konfidence-ui:${COMMIT_SHA}"] : [],
  )
}
