gen-update-package:
	go get github.com/magic-lib/go-plat-cache@master
	go get github.com/magic-lib/go-plat-startupcfg@master
	go get github.com/magic-lib/go-plat-utils@master
	go mod tidy