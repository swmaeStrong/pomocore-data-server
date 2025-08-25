package domain

type CategoryPattern struct {
	ID             string
	Category       string
	Priority       int
	AppPatterns    []string
	DomainPatterns []string
}

func NewCategoryPattern(
	category string,
	priority int,
	appPatterns []string,
	domainPatterns []string,
) *CategoryPattern {
	return &CategoryPattern{
		Category:       category,
		Priority:       priority,
		AppPatterns:    appPatterns,
		DomainPatterns: domainPatterns,
	}
}

func (c *CategoryPattern) AddAppPattern(pattern string) {
	if !c.containsAppPattern(pattern) {
		c.AppPatterns = append(c.AppPatterns, pattern)
	}
}

func (c *CategoryPattern) AddDomainPattern(pattern string) {
	if !c.containsDomainPattern(pattern) {
		c.DomainPatterns = append(c.DomainPatterns, pattern)
	}
}

func (c *CategoryPattern) containsAppPattern(pattern string) bool {
	for _, p := range c.AppPatterns {
		if p == pattern {
			return true
		}
	}
	return false
}

func (c *CategoryPattern) containsDomainPattern(pattern string) bool {
	for _, p := range c.DomainPatterns {
		if p == pattern {
			return true
		}
	}
	return false
}
