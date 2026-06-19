terraform {
  required_version = ">= 1.6.0"
}

locals {
  environment = "production"
  stage       = "stage-1-probe"
}
