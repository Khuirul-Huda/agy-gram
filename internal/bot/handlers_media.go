package bot

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

// HandlePhoto handles incoming photo uploads by downloading to workspace and triggering AGY analysis.
func (h *Handler) HandlePhoto(c tele.Context) error {
	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	caption := strings.TrimSpace(c.Message().Caption)
	if caption == "" {
		caption = "Tolong analisis dan jelaskan gambar/screenshot ini secara detail."
	}

	ws := h.session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}

	// Prepare uploads directory in workspace
	uploadDir := filepath.Join(ws, "uploads")
	_ = os.MkdirAll(uploadDir, 0755)

	fileName := fmt.Sprintf("photo_%d.jpg", time.Now().Unix())
	destPath := filepath.Join(uploadDir, fileName)

	// Download file from Telegram
	rc, err := c.Bot().File(&photo.File)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal mengunduh foto dari Telegram: %v", err))
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal menyimpan file: %v", err))
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal menulis file: %v", err))
	}

	prompt := fmt.Sprintf("%s\n\n[File Gambar Terlampir]: %s", caption, destPath)
	_ = c.Send(fmt.Sprintf("📸 Foto berhasil diterima & disimpan di <code>%s</code>. Memulai analisis...", destPath), tele.ModeHTML)

	return h.executePrompt(c, prompt)
}

// HandleDocument handles incoming document/file uploads.
func (h *Handler) HandleDocument(c tele.Context) error {
	doc := c.Message().Document
	if doc == nil {
		return nil
	}

	caption := strings.TrimSpace(c.Message().Caption)
	if caption == "" {
		caption = fmt.Sprintf("Analisis dokumen/kode berikut: %s", doc.FileName)
	}

	ws := h.session.GetWorkspace()
	if ws == "" {
		ws = "/home/huda"
	}

	uploadDir := filepath.Join(ws, "uploads")
	_ = os.MkdirAll(uploadDir, 0755)

	destPath := filepath.Join(uploadDir, doc.FileName)

	rc, err := c.Bot().File(&doc.File)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal mengunduh dokumen dari Telegram: %v", err))
	}
	defer rc.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal menyimpan file dokumen: %v", err))
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return c.Send(fmt.Sprintf("❌ Gagal menulis dokumen: %v", err))
	}

	prompt := fmt.Sprintf("%s\n\n[File Dokumen Terlampir]: %s", caption, destPath)
	_ = c.Send(fmt.Sprintf("📄 Dokumen <code>%s</code> berhasil disimpan. Memproses...", doc.FileName), tele.ModeHTML)

	return h.executePrompt(c, prompt)
}

// HandleFileDownloadCallback sends a modified/generated file directly to Telegram chat.
func (h *Handler) HandleFileDownloadCallback(c tele.Context) error {
	_ = c.Respond()
	cb := c.Callback()
	if cb == nil || cb.Data == "" {
		return nil
	}

	filePath := cb.Data
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Send(fmt.Sprintf("❌ File tidak ditemukan: <code>%s</code>", filePath), tele.ModeHTML)
	}

	doc := &tele.Document{
		File:     tele.FromDisk(filePath),
		FileName: filepath.Base(filePath),
		Caption:  fmt.Sprintf("📥 Unduhan file: <code>%s</code>", filepath.Base(filePath)),
	}

	return c.Send(doc, tele.ModeHTML)
}
