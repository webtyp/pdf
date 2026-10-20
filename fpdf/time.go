package fpdf

import (
	"webtyp.com/time"
)

// SetDefaultCreationDate sets the default value of the document creation date
// that will be used when initializing a new Fpdf instance. See
// SetCreationDate() for more details.
func SetDefaultCreationDate(tm time.Unix) {
	gl.creationDate = tm
}

// SetDefaultModificationDate sets the default value of the document modification date
// that will be used when initializing a new Fpdf instance. See
// SetCreationDate() for more details.
func SetDefaultModificationDate(tm time.Unix) {
	gl.modDate = tm
}

// GetCreationDate returns the document's internal CreationDate value.
func (f *Fpdf) GetCreationDate() time.Unix {
	return f.creationDate
}

// SetCreationDate fixes the document's internal CreationDate value. By
// default, the time when the document is generated is used for this value.
// This method is typically only used for testing purposes to facilitate PDF
// comparison. Specify a zero-value time to revert to the default behavior.
func (f *Fpdf) SetCreationDate(tm time.Unix) {
	f.creationDate = tm
}

// GetModificationDate returns the document's internal ModDate value.
func (f *Fpdf) GetModificationDate() time.Unix {
	return f.modDate
}

// SetModificationDate fixes the document's internal ModDate value.
// See `SetCreationDate` for more details.
func (f *Fpdf) SetModificationDate(tm time.Unix) {
	f.modDate = tm
}

// timeOrNow returns the current instant when tm is zero.
func timeOrNow(tm time.Unix) time.Unix {
	if tm == 0 {
		return time.NowUnix()
	}
	return tm
}

// formatPDFDate writes a PDF date, "D:YYYYMMDDHHmmss" (UTC).
func formatPDFDate(tm time.Unix) string {
	return "D:" + time.FormatCompact(timeOrNow(tm))
}
