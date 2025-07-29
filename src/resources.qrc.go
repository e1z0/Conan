package main

//go:generate miqt-rcc -Input "src/resources.qrc" -OutputGo "resources.qrc.go" -OutputRcc "resources.qrc.rcc" -RccBinary "/opt/homebrew/Cellar/qt@5/5.15.16_2/bin/rcc"

import (
	"embed"

	"github.com/mappu/miqt/qt"
)

//go:embed resources.qrc.rcc
var _resourceRcc []byte

func init() {
	_ = embed.FS{}
	qt.QResource_RegisterResourceWithRccData(&_resourceRcc[0])
}
