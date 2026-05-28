
variable "TAG" {
  default = "dev"
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
  tags       = ["${REGISTRY}/star-operator:${TAG}"]
  args       = { OPERATOR_NAME = "star" }
}

target "galaxy-operator" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/galaxy-operator:${TAG}"]
  args       = { OPERATOR_NAME = "galaxy" }
}
