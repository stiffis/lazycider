package kitty

func ComputeCoverPlacementSize(termWidth, panelW, panelH, imgW, imgH int) (placeW, placeH int) {
	if termWidth <= 0 || panelW <= 0 || panelH <= 0 {
		return 0, 0
	}

	const (
		cellPixelW = 10
		cellPixelH = 20
	)

	targetW := (termWidth * 20) / 100
	if targetW < 1 {
		targetW = 1
	}
	if targetW > panelW {
		targetW = panelW
	}

	placeW = targetW
	if imgW <= 0 || imgH <= 0 {
		placeH = (placeW*cellPixelW + cellPixelH/2) / cellPixelH
	} else {
		numerator := placeW * imgH * cellPixelW
		denominator := imgW * cellPixelH
		placeH = (numerator + denominator/2) / denominator
	}

	if placeH < 1 {
		placeH = 1
	}

	if placeH > panelH {
		placeH = panelH
		if imgW > 0 && imgH > 0 {
			placeW = (placeH*imgW + imgH/2) / imgH
			if placeW < 1 {
				placeW = 1
			}
			if placeW > panelW {
				placeW = panelW
			}
		}
	}

	return placeW, placeH
}
