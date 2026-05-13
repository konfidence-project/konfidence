variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["galaxy-vector-promotion-controller"]
}

target "galaxy-vector-promotion-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/galaxy-vector-promotion-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
