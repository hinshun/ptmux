package terminal

import "github.com/hinshun/vt10x"

type mouseSupport struct {
	mode vt10x.ModeFlag
}

func (ms *mouseSupport) MouseEnabled() bool {
	return (ms.mode&vt10x.ModeMouseButton != 0) ||
		(ms.mode&vt10x.ModeMouseMotion != 0) ||
		(ms.mode&vt10x.ModeMouseMany != 0)
}

func (ms *mouseSupport) MouseIsSgr() bool {
	return ms.mode&vt10x.ModeMouseSgr != 0
}

func (ms *mouseSupport) MouseReportButton() bool {
	return ms.mode&vt10x.ModeMouseMotion != 0
}

func (ms *mouseSupport) MouseReportAny() bool {
	return ms.mode&vt10x.ModeMouseMany != 0
}
