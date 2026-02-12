package util

import "strings"

func PreprocessArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	out := make([]string, 0, len(args))
	out = append(out, args[0])

	i := 1
	for i < len(args) {
		arg := args[i]

		if arg == "--between" && i+3 < len(args) {
			field := args[i+1]
			start := args[i+2]
			end := args[i+3]
			if !strings.HasPrefix(field, "-") && !strings.HasPrefix(start, "-") && !strings.HasPrefix(end, "-") {
				out = append(out, "--between", field+"="+start+":"+end)
				i += 4
				continue
			}
		}

		if arg == "--in" && i+2 < len(args) {
			field := args[i+1]
			values := args[i+2]
			if !strings.HasPrefix(field, "-") && !strings.HasPrefix(values, "-") {
				out = append(out, "--in", field+"="+values)
				i += 3
				continue
			}
		}

		out = append(out, arg)
		i++
	}

	return out
}
