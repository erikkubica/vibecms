package coreapi

import (
	"context"
	"fmt"
)

func (c *coreImpl) UploadMedia(_ context.Context, req MediaUploadRequest) (*MediaFile, error) {
	if c.mediaSvc == nil {
		return nil, NewInternal("media service not configured")
	}

	mf, err := c.mediaSvc.Upload(req.Body, req.Filename, req.MimeType, 0)
	if err != nil {
		return nil, NewInternal(fmt.Sprintf("media upload failed: %v", err))
	}

	return &MediaFile{
		ID:        mf.ID,
		Filename:  mf.OriginalName,
		MimeType:  mf.MimeType,
		Size:      mf.Size,
		URL:       mf.URL,
		CreatedAt: mf.CreatedAt,
	}, nil
}

func (c *coreImpl) GetMedia(_ context.Context, id uint) (*MediaFile, error) {
	if c.mediaSvc == nil {
		return nil, NewInternal("media service not configured")
	}

	mf, err := c.mediaSvc.GetByID(id)
	if err != nil {
		return nil, NewNotFound("media_file", id)
	}

	return &MediaFile{
		ID:        mf.ID,
		Filename:  mf.OriginalName,
		MimeType:  mf.MimeType,
		Size:      mf.Size,
		URL:       mf.URL,
		CreatedAt: mf.CreatedAt,
	}, nil
}

func (c *coreImpl) QueryMedia(_ context.Context, query MediaQuery) ([]*MediaFile, error) {
	if c.mediaSvc == nil {
		return nil, NewInternal("media service not configured")
	}

	files, _, err := c.mediaSvc.List(query.MimeType, query.Search, query.Limit, query.Offset)
	if err != nil {
		return nil, NewInternal(fmt.Sprintf("media query failed: %v", err))
	}

	result := make([]*MediaFile, len(files))
	for i, mf := range files {
		result[i] = &MediaFile{
			ID:        mf.ID,
			Filename:  mf.OriginalName,
			MimeType:  mf.MimeType,
			Size:      mf.Size,
			URL:       mf.URL,
			CreatedAt: mf.CreatedAt,
		}
	}

	return result, nil
}

func (c *coreImpl) DeleteMedia(_ context.Context, id uint) error {
	if c.mediaSvc == nil {
		return NewInternal("media service not configured")
	}

	if err := c.mediaSvc.Delete(id); err != nil {
		return NewInternal(fmt.Sprintf("media delete failed: %v", err))
	}

	return nil
}
