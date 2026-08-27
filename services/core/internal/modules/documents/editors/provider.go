package editors

import (
	"context"
	"errors"
)

type ProviderID string

const (
	ProviderNative             ProviderID = "native"
	ProviderCollabora          ProviderID = "collabora"
	ProviderOnlyOffice         ProviderID = "onlyoffice"
	ProviderLibreOfficeDesktop ProviderID = "libreoffice-desktop"
	ProviderGoogleDocs         ProviderID = "google-docs"
	ProviderGoogleSheets       ProviderID = "google-sheets"
	ProviderGoogleSlides       ProviderID = "google-slides"
)

type Capability string

const (
	CapabilityCreate          Capability = "create"
	CapabilityEdit            Capability = "edit"
	CapabilityView            Capability = "view"
	CapabilityConvert         Capability = "convert"
	CapabilityExport          Capability = "export"
	CapabilityCollaborative   Capability = "collaborative"
	CapabilityBrowser         Capability = "browser"
	CapabilityDesktop         Capability = "desktop"
	CapabilityOffline         Capability = "offline"
	CapabilityExternal        Capability = "external"
	CapabilityExternalProcess Capability = "external-process"
)

type DocumentFamily string

const (
	FamilyNative       DocumentFamily = "native"
	FamilyText         DocumentFamily = "text"
	FamilySpreadsheet  DocumentFamily = "spreadsheet"
	FamilyPresentation DocumentFamily = "presentation"
	FamilyPDF          DocumentFamily = "pdf"
)

type StorageMode string

const (
	StorageManaged  StorageMode = "managed"
	StorageExternal StorageMode = "external"
)

type Capabilities struct {
	Operations []Capability     `json:"operations"`
	Families   []DocumentFamily `json:"families"`
}

func (capabilities Capabilities) Supports(required ...Capability) bool {
	for _, requirement := range required {
		found := false
		for _, capability := range capabilities.Operations {
			if capability == requirement {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (capabilities Capabilities) SupportsFamily(family DocumentFamily) bool {
	for _, supported := range capabilities.Families {
		if supported == family {
			return true
		}
	}
	return false
}

type Availability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

type CreateDocumentRequest struct {
	ActorID     string
	WorkspaceID string
	Title       string
	Family      DocumentFamily
}

type CreateDocumentResult struct {
	StorageMode      StorageMode
	NativeFormat     string
	ExternalProvider ProviderID
	ExternalFileID   string
	ExternalURL      string
}

type OpenDocumentRequest struct {
	ActorID    string
	DocumentID string
	CanRead    bool
	CanEdit    bool
}

type OpenDocumentResult struct {
	Mode     string
	URL      string
	ReadOnly bool
}

var (
	ErrProviderNotFound    = errors.New("editor provider not found")
	ErrProviderUnavailable = errors.New("editor provider unavailable")
	ErrCapabilityMissing   = errors.New("editor provider does not support the required capability")
	ErrOperationDeferred   = errors.New("editor provider operation is not implemented")
)

type Provider interface {
	ID() ProviderID
	Capabilities(context.Context) Capabilities
	Available(context.Context) Availability
	Create(context.Context, CreateDocumentRequest) (*CreateDocumentResult, error)
	Open(context.Context, OpenDocumentRequest) (*OpenDocumentResult, error)
}

type Status struct {
	ID           ProviderID       `json:"id"`
	Available    bool             `json:"available"`
	Reason       string           `json:"reason,omitempty"`
	Capabilities []Capability     `json:"capabilities"`
	Families     []DocumentFamily `json:"families"`
}
