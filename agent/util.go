package agent

func deployPointers(deploys ...Deploy) []*Deploy {
	out := make([]*Deploy, 0, len(deploys))
	for _, a := range deploys {
		tmp := a
		out = append(out, &tmp)
	}
	return out
}

func deploysFirstOrDefault(def Deploy, deploys ...Deploy) Deploy {
	if len(deploys) > 0 {
		return deploys[0]
	}

	return def
}

func deployArchives(deploys ...Deploy) []*Archive {
	results := make([]*Archive, 0, len(deploys))
	for _, a := range deploys {
		results = append(results, a.Archive)
	}
	return results
}
