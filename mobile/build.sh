#!/bin/bash
gomobile bind -target=ios github.com/mit-dci/go-bverify/mobile
rm -rf ~/Documents/BVerifyClient/iOS/BVerifyClient/BVerifyClient/Mobile.framework && cp -R Mobile.framework ~/Documents/BVerifyClient/iOS/BVerifyClient/BVerifyClient
