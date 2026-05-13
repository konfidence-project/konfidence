variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["galaxy-stage-configuration-controller"]
}

target "galaxy-stage-configuration-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/galaxy-stage-configuration-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
