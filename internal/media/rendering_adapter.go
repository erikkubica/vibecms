package media

import "vibecms/internal/rendering"

// sizeRegistryAdapter adapts SizeRegistry to rendering.ImageSizeProvider.
type sizeRegistryAdapter struct {
	registry *SizeRegistry
}

// AsImageSizeProvider wraps a SizeRegistry to satisfy rendering.ImageSizeProvider.
func AsImageSizeProvider(r *SizeRegistry) rendering.ImageSizeProvider {
	return &sizeRegistryAdapter{registry: r}
}

func (a *sizeRegistryAdapter) GetAll() []rendering.ImageSizeInfo {
	sizes := a.registry.GetAll()
	out := make([]rendering.ImageSizeInfo, len(sizes))
	for i, s := range sizes {
		out[i] = rendering.ImageSizeInfo{
			Name:   s.Name,
			Width:  s.Width,
			Height: s.Height,
		}
	}
	return out
}

func (a *sizeRegistryAdapter) GetByName(name string) (rendering.ImageSizeInfo, bool) {
	s, ok := a.registry.Get(name)
	if !ok {
		return rendering.ImageSizeInfo{}, false
	}
	return rendering.ImageSizeInfo{
		Name:   s.Name,
		Width:  s.Width,
		Height: s.Height,
	}, true
}
