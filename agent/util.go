package agent

func deployPointers(deploys ...Deploy) []*Deploy {
	out := make([]*Deploy, 0, len(deploys))
	for _, a := range deploys {
		tmp := a
		out = append(out, &tmp)
	}
	return out
}
