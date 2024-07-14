package model

func (m *Model) onNextSearch() {
	origSearch := m.search
	m.search = (m.search + 1) % len(m.searches)
	for m.search != origSearch {
		if m.searches[m.search].pattern != "" {
			break
		}
		m.search = (m.search + 1) % len(m.searches)
	}
}

func (m *Model) onPrevSearch() {
	origSearch := m.search
	m.search = (len(m.searches) + m.search - 1) % len(m.searches)
	for m.search != origSearch {
		if m.searches[m.search].pattern != "" {
			break
		}
		m.search = (len(m.searches) + m.search - 1) % len(m.searches)
	}
}
