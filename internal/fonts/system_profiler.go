package fonts

type spFontsDataTypeObjectJSON struct {
	SPFontsDataType []spFontsDataTypeJSON
}
type spFontsDataTypeJSON struct {
	Typefaces []spTypeface `json:"typefaces"`
}
type spTypeface struct {
	Family string `json:"family"`
	Style  string `json:"style"`
}
