#!/bin/bash

ACCESS_TOKEN=XXX

export ACCESS_TOKEN

ginkgo -r -skipPackage=real_api
