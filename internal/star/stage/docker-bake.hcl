variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["landscape-stage-controller"]
}

target "landscape-stage-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/landscape-stage-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
