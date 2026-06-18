terraform {
  required_version = ">= 1.6.0"
}

locals {
  environment = "dev"
  stage       = "stage-0-bootstrap"
}
