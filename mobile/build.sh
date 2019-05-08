#!/bin/bash
if [[ "$OSTYPE" == "darwin"* ]]; then
	gomobile bind -target=ios github.com/mit-dci/go-bverify/mobile
	rm -rf ~/bverify-mobile/iOS/BVerifyClient/Mobile.framework && cp -R Mobile.framework ~/bverify-mobile/iOS/BVerifyClient
fi

gomobile bind -target=android github.com/mit-dci/go-bverify/mobile
rm -rf ~/src/bverify-mobile/Android/mobile/mobile.aar && cp mobile.aar ~/src/bverify-mobile/Android/mobile/mobile.aar

