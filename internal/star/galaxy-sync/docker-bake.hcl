variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["landscape-gcp-sync-controller"]
}

target "landscape-gcp-sync-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/landscape-gcp-sync-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
