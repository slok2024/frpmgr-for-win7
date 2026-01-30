package ui

import (
	"image"

	"github.com/lxn/walk"

	"github.com/koho/frpmgr/pkg/consts"
	"github.com/koho/frpmgr/pkg/res"
)

var cachedIconsForWidthAndId = make(map[widthAndId]*walk.Icon)

func loadIcon(id res.Icon, size int) (icon *walk.Icon) {
	icon = cachedIconsForWidthAndId[widthAndId{size, id}]
	if icon != nil {
		return
	}
	var err error
	if id.Dll == "" {
		icon, err = walk.NewIconFromResourceIdWithSize(id.Index, walk.Size{Width: size, Height: size})
	} else {
		icon, err = walk.NewIconFromSysDLLWithSize(id.Dll, id.Index, size)
	}

	// 【核心修复】如果加载失败，绝对不能返回 nil。
	// 使用 walk 系统内建图标作为保底，防止 walk 库在调用 Icon.Size() 时崩溃。
	if err != nil || icon == nil {
		icon = walk.IconError()
		if icon == nil {
			icon = walk.IconInformation()
		}
	}

	if icon != nil {
		cachedIconsForWidthAndId[widthAndId{size, id}] = icon
	}
	return
}

type widthAndId struct {
	width int
	icon  res.Icon
}

type widthAndConfigState struct {
	width int
	state consts.ConfigState
}

var cachedIconsForWidthAndConfigState = make(map[widthAndConfigState]*walk.Icon)

func iconForConfigState(state consts.ConfigState, size int) (icon *walk.Icon) {
	icon = cachedIconsForWidthAndConfigState[widthAndConfigState{size, state}]
	if icon != nil {
		return
	}
	switch state {
	case consts.ConfigStateStarted:
		icon = loadIcon(res.IconStateRunning, size)
	case consts.ConfigStateStopped, consts.ConfigStateUnknown:
		icon = loadIcon(res.IconStateStopped, size)
	default:
		icon = loadIcon(res.IconStateWorking, size)
	}
	if icon != nil {
		cachedIconsForWidthAndConfigState[widthAndConfigState{size, state}] = icon
	}
	return
}

type widthAndProxyState struct {
	width int
	state consts.ProxyState
}

var cachedIconsForWidthAndProxyState = make(map[widthAndProxyState]*walk.Icon)

func iconForProxyState(state consts.ProxyState, size int) (icon *walk.Icon) {
	icon = cachedIconsForWidthAndProxyState[widthAndProxyState{size, state}]
	if icon != nil {
		return
	}
	switch state {
	case consts.ProxyStateRunning:
		icon = loadIcon(res.IconProxyRunning, size)
	case consts.ProxyStateError:
		icon = loadIcon(res.IconProxyError, size)
	default:
		icon = loadIcon(res.IconStateStopped, size)
	}
	if icon != nil {
		cachedIconsForWidthAndProxyState[widthAndProxyState{size, state}] = icon
	}
	return
}

func loadLogoIcon(size int) *walk.Icon {
	return loadIcon(res.IconLogo, size)
}

func loadShieldIcon(size int) (icon *walk.Icon) {
	icon = loadIcon(res.IconNewVersion1, size)
	if icon == nil {
		icon = loadIcon(res.IconNewVersion2, size)
	}
	return
}

func drawCopyIcon(canvas *walk.Canvas, color walk.Color) error {
	dpi := canvas.DPI()
	point := func(x, y int) walk.Point {
		return walk.PointFrom96DPI(walk.Point{X: x, Y: y}, dpi)
	}
	rectangle := func(x, y, width, height int) walk.Rectangle {
		return walk.RectangleFrom96DPI(walk.Rectangle{X: x, Y: y, Width: width, Height: height}, dpi)
	}

	brush, err := walk.NewSolidColorBrush(color)
	if err != nil {
		return err
	}
	defer brush.Dispose()

	pen, err := walk.NewGeometricPen(walk.PenSolid|walk.PenInsideFrame|walk.PenCapSquare|walk.PenJoinMiter, 2, brush)
	if err != nil {
		return err
	}
	defer pen.Dispose()

	bounds := rectangle(5, 5, 8, 9)
	startPoint := point(3, 3)
	if penWidth := walk.IntFrom96DPI(pen.Width(), dpi); bounds.X-(startPoint.X+(penWidth-1)/2) < 2 {
		bounds.X++
		bounds.Y++
	}

	if err = canvas.DrawRectanglePixels(pen, bounds); err != nil {
		return err
	}
	if err = canvas.DrawLinePixels(pen, startPoint, point(9, 3)); err != nil {
		return err
	}
	if err = canvas.DrawLinePixels(pen, startPoint, point(3, 10)); err != nil {
		return err
	}
	return nil
}

func flipIcon(id res.Icon, size int) *walk.PaintFuncImage {
	size96dpi := walk.Size{Width: size, Height: size}
	return walk.NewPaintFuncImagePixels(size96dpi, func(canvas *walk.Canvas, bounds walk.Rectangle) error {
		size := walk.SizeFrom96DPI(size96dpi, canvas.DPI())
		targetIcon := loadIcon(id, size.Width)
		if targetIcon == nil {
			return nil
		}
		bitmap, err := walk.NewBitmapFromIconForDPI(targetIcon, size, canvas.DPI())
		if err != nil {
			return err
		}
		defer bitmap.Dispose()
		img, err := bitmap.ToImage()
		if err != nil {
			return err
		}
		rotated := image.NewRGBA(img.Rect)
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				rotated.Set(img.Bounds().Max.X-x-1, img.Bounds().Max.Y-y-1, img.At(x, y))
			}
		}
		bitmap, err = walk.NewBitmapFromImageForDPI(rotated, canvas.DPI())
		if err != nil {
			return err
		}
		defer bitmap.Dispose()
		return canvas.DrawImageStretchedPixels(bitmap, bounds)
	})
}
