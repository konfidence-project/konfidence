variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["gcp-vector-promotion-controller"]
}

target "gcp-vector-promotion-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/gcp-vector-promotion-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
