variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["gcp-stage-configuration-controller"]
}

target "gcp-stage-configuration-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/gcp-stage-configuration-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
