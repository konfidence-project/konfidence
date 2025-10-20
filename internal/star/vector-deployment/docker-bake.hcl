variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["landscape-vector-deployment-controller"]
}

target "landscape-vector-deployment-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/landscape-vector-deployment-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
