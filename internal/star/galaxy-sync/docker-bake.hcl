variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["star-galaxy-sync-controller"]
}

target "star-galaxy-sync-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/star-galaxy-sync-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
