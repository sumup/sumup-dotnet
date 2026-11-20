package generator

import (
	"cmp"
	"fmt"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type resourceParent struct {
	tag   string
	label string
}

func readResourceParents(doc *v3.Document) (map[string]resourceParent, error) {
	if doc.Extensions == nil {
		return nil, nil
	}
	node, ok := doc.Extensions.Get("x-tagGroups")
	if !ok {
		return nil, nil
	}
	var groups []struct {
		Name string   `yaml:"name"`
		Tags []string `yaml:"tags"`
	}
	if err := node.Decode(&groups); err != nil {
		return nil, fmt.Errorf("decode x-tagGroups: %w", err)
	}
	labels := make(map[string]string, len(doc.Tags))
	for _, tag := range doc.Tags {
		var displayName string
		if tag.Extensions != nil {
			if label, ok := tag.Extensions.Get("x-displayName"); ok {
				if err := label.Decode(&displayName); err != nil {
					return nil, fmt.Errorf("decode x-displayName for %s: %w", tag.Name, err)
				}
			}
		}
		labels[tag.Name] = cmp.Or(displayName, tag.Name)
	}
	parents := map[string]resourceParent{}
	for _, group := range groups {
		if labels[group.Name] == "" {
			continue
		}
		for _, child := range group.Tags {
			if child == group.Name || labels[child] == "" || parents[child].tag != "" {
				continue
			}
			for ancestor := group.Name; ancestor != ""; ancestor = parents[ancestor].tag {
				if ancestor == child {
					return nil, fmt.Errorf("tag hierarchy contains a cycle at %s", child)
				}
			}
			parents[child] = resourceParent{tag: group.Name, label: labels[child]}
		}
	}
	return parents, nil
}
