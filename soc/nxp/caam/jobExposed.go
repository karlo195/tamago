package caam

func (hw *CAAM) Job(hdr *Header, jd []byte) (err error) {
	return hw.job(hdr,jd)
}
