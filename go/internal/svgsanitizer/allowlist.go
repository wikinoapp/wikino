package svgsanitizer

// allowedElementsは、残すSVGの要素。ここに無い要素は、タグだけを落として子を残す
// (droppedElementsと、SVG以外の名前空間の要素を除く)
var allowedElements = toSet(
	"svg", "g", "path", "rect", "circle", "ellipse", "line", "polyline", "polygon", "text", "tspan",
	"defs", "clipPath", "mask", "pattern", "linearGradient", "radialGradient", "stop",
	"use", "symbol", "marker", "image", "title", "desc", "metadata",
	// 基本の要素に加えて許す要素。switchは、条件に合う最初の子だけを描くため、タグだけを落として
	// 子をすべて描くと、言語ごとの文字が重なる
	"switch",
)

// droppedElementsは、子と中身ごと落とすSVGの要素。スクリプトとスタイルシートのほか、HTMLを埋め込む
// foreignObjectも落とす。タグだけを落とすと、HTMLの文字が図の中に残るため
var droppedElements = toSet("script", "style", "foreignObject")

// commonAttributesは、残すすべての要素に許す属性
var commonAttributes = toSet(
	"id", "class", "style", "fill", "stroke", "stroke-width", "stroke-linecap",
	"stroke-linejoin", "stroke-miterlimit", "stroke-dasharray",
	"stroke-dashoffset", "stroke-opacity", "fill-opacity", "opacity",
	"transform",
	// 基本の属性に加えて許す、見た目だけを決める属性。Figma・Inkscape・draw.ioの書き出しが使い、
	// 落とすと切り抜き・塗りの規則・文字の書体が変わる。文字の属性は子に引き継がれるため、gやtspanにも許す
	"clip-path", "clip-rule", "mask", "fill-rule", "display", "visibility",
	"font-family", "font-size", "font-weight", "text-anchor", "font-style", "font-variant", "letter-spacing", "word-spacing", "text-decoration",
	"dominant-baseline", "marker-start", "marker-mid", "marker-end", "paint-order", "vector-effect",
	"systemLanguage", "requiredFeatures", "requiredExtensions",
)

// elementAttributesは、要素ごとに許す属性。`xlink:href` はXLinkの名前空間のhref属性を表す。
// グラデーションとパターンのhref (ほかの定義を引き継ぐ)・symbolとmarkerの属性も許す
var elementAttributes = map[string]map[string]struct{}{
	"svg":            toSet("viewBox", "width", "height", "preserveAspectRatio"),
	"image":          toSet("x", "y", "width", "height", "xlink:href", "href", "preserveAspectRatio"),
	"text":           toSet("x", "y", "dx", "dy"),
	"tspan":          toSet("x", "y", "dx", "dy"),
	"linearGradient": toSet("x1", "y1", "x2", "y2", "gradientUnits", "gradientTransform", "spreadMethod", "xlink:href", "href"),
	"radialGradient": toSet("cx", "cy", "r", "fx", "fy", "gradientUnits", "gradientTransform", "spreadMethod", "xlink:href", "href"),
	"stop":           toSet("offset", "stop-color", "stop-opacity"),
	"pattern":        toSet("x", "y", "width", "height", "patternUnits", "patternContentUnits", "patternTransform", "viewBox", "preserveAspectRatio", "xlink:href", "href"),
	"symbol":         toSet("viewBox", "preserveAspectRatio"),
	"marker":         toSet("viewBox", "preserveAspectRatio", "refX", "refY", "markerWidth", "markerHeight", "markerUnits", "orient"),
	"clipPath":       toSet("clipPathUnits"),
	"mask":           toSet("x", "y", "width", "height", "maskUnits", "maskContentUnits"),
	"use":            toSet("x", "y", "width", "height", "xlink:href", "href"),
	"path":           toSet("d"),
	"rect":           toSet("x", "y", "width", "height", "rx", "ry"),
	"circle":         toSet("cx", "cy", "r"),
	"ellipse":        toSet("cx", "cy", "rx", "ry"),
	"line":           toSet("x1", "y1", "x2", "y2"),
	"polyline":       toSet("points"),
	"polygon":        toSet("points"),
}

// allowedFunctionsは、属性の値の中に許す関数 (CSSの関数とtransformの関数)。urlは、文書の中の要素を
// 指すものだけを許す (allowedFunctionCallで確かめる)。ここに無い関数は、image-setのように外部の
// リソースを読みうるため、その属性を落とす
var allowedFunctions = toSet(
	"url", "rgb", "rgba", "hsl", "hsla", "hwb", "lab", "lch", "oklab", "oklch", "color",
	"calc", "min", "max", "clamp", "var",
	"matrix", "translate", "translatex", "translatey", "scale", "scalex", "scaley",
	"rotate", "skewx", "skewy",
)

func toSet(names ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}
