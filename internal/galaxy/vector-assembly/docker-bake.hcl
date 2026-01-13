variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["gcp-vector-assembly-controller"]
}

target "gcp-vector-assembly-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/gcp-vector-assembly-controller:${TAG}"]

  secret = ["id=gh_token,env=GH_TOKEN"]
}
