package server

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

func TestAvatarKeepsGIF(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret-pass", "Owner")
	other := mustAccount(t, e, "other@example.com", "secret-pass", "Other")

	raw := twoFrameGIF(t)
	saved := postFile(t, e, "/account/avatar", "move.gif", "image/gif", raw, owner)
	if saved.Code != http.StatusOK || !strings.Contains(saved.Body.String(), `"hasAvatar":true`) {
		t.Fatalf("save %d %s", saved.Code, saved.Body.String())
	}
	got := getJSON(t, e, "/accounts/Owner/avatar", nil)
	if got.Code != http.StatusOK || got.Header().Get("Content-Type") != "image/gif" || got.Header().Get("X-Content-Type-Options") != "nosniff" || !bytes.Equal(got.Body.Bytes(), raw) {
		t.Fatalf("gif %d %s %d bytes", got.Code, got.Header().Get("Content-Type"), got.Body.Len())
	}
	page := getJSON(t, e, "/accounts/Owner", nil)
	if !strings.Contains(page.Body.String(), `"hasAvatar":true`) || !strings.Contains(page.Body.String(), `"avatarUpdated"`) {
		t.Fatalf("page %s", page.Body.String())
	}
	if stolen := postFile(t, e, "/account/avatar", "own.png", "image/png", onePixelPNG(t), other); stolen.Code != http.StatusOK {
		t.Fatalf("other avatar %d %s", stolen.Code, stolen.Body.String())
	}
	again := getJSON(t, e, "/accounts/Owner/avatar", nil)
	if !bytes.Equal(again.Body.Bytes(), raw) {
		t.Fatal("owner gif changed")
	}
	cleared := deleteCookie(t, e, "/account/avatar", owner)
	if cleared.Code != http.StatusOK || !strings.Contains(cleared.Body.String(), `"hasAvatar":false`) {
		t.Fatalf("clear %d %s", cleared.Code, cleared.Body.String())
	}
	missing := getJSON(t, e, "/accounts/Owner/avatar", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing %d %s", missing.Code, missing.Body.String())
	}
}

func TestAvatarRejects(t *testing.T) {
	e := newServer(t)
	owner := mustAccount(t, e, "owner@example.com", "secret-pass", "Owner")
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{name: "html", body: []byte("<html>no</html>"), want: "Подойдёт PNG, JPEG или GIF."},
		{name: "wide", body: widePNG(t), want: "1\u202f024"},
		{name: "huge", body: bytes.Repeat([]byte{0}, maxAvatarBytes+1), want: "2 МБ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postFile(t, e, "/account/avatar", "file.bin", "application/octet-stream", tt.body, owner)
			if got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), tt.want) {
				t.Fatalf("%d %s", got.Code, got.Body.String())
			}
		})
	}
	anon := postFile(t, e, "/account/avatar", "a.gif", "image/gif", twoFrameGIF(t), nil)
	if anon.Code != http.StatusUnauthorized {
		t.Fatalf("anon %d %s", anon.Code, anon.Body.String())
	}
}

func postFile(t *testing.T, h http.Handler, path, filename, contentType string, body []byte, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="` + filename + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func onePixelPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func widePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, maxAvatarEdge+1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func twoFrameGIF(t *testing.T) []byte {
	t.Helper()
	palette := color.Palette{color.Black, color.White}
	first := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	second := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	second.SetColorIndex(0, 0, 1)
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, &gif.GIF{
		Image: []*image.Paletted{first, second},
		Delay: []int{10, 10},
	}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
