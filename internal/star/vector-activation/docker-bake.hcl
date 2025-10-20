variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["landscape-vector-activation-controller"]
}

target "landscape-vector-activation-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/landscape-vector-activation-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
