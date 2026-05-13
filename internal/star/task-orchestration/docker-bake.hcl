variable "TAG" {
  default = "dev"
}

variable "REGISTRY" {
  default = "ghcr.io/konfidence-project"
}

group "default" {
  targets = ["landscape-task-orchestration-controller"]
}

target "landscape-task-orchestration-controller" {
  context    = "."
  dockerfile = "Dockerfile"
  platforms  = ["linux/amd64", "linux/arm64"]
  tags       = ["${REGISTRY}/landscape-task-orchestration-controller:${TAG}"]
  
  secret = ["id=gh_token,env=GH_TOKEN"]
}
