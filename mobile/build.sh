#!/bin/bash
gomobile bind -target=ios github.com/mit-dci/go-bverify/mobile
rm -rf ~/bverify-mobile/iOS/BVerifyClient/Mobile.framework && cp -R Mobile.framework ~/bverify-mobile/iOS/BVerifyClient
