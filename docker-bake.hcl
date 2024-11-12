variable "BUILD_TAG" {
  default = ""
}

target "build-unix" {
  dockerfile = "Dockerfile"
  tags       = ["${BUILD_TAG}-unix"]
  output     = ["type=registry", "compression=gzip", "compression-level=9", "force-compression=true"]
  args = { BASE_IMAGE = "scratch" }
  platforms  = [
    "freebsd/amd64",
    "freebsd/arm64",
    "freebsd/386",
    "freebsd/arm",
    "linux/amd64",
    "linux/arm64",
    "linux/arm",
    "linux/s390x",
    "linux/ppc64le",
    "linux/386",
  ]
}
