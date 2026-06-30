package qa

import (
	"fmt"
	"math/rand"
)

// GenerateSuggestions generates 3 suggested questions based on column names.
// Each suggestion references at least one actual column name.
func GenerateSuggestions(headers []string) []string {
	if len(headers) == 0 {
		return []string{
			"這份資料有哪些特徵？",
			"資料的整體趨勢是什麼？",
			"有哪些異常值得注意？",
		}
	}

	suggestions := make([]string, 0, 3)

	// Template patterns (each references at least one column)
	type templateFunc func(cols []string) string

	singleColTemplates := []templateFunc{
		func(cols []string) string { return fmt.Sprintf("哪個%s的數值最高？", cols[0]) },
		func(cols []string) string { return fmt.Sprintf("%s的分佈情況如何？", cols[0]) },
		func(cols []string) string { return fmt.Sprintf("%s有哪些異常值？", cols[0]) },
		func(cols []string) string { return fmt.Sprintf("%s的平均值是多少？", cols[0]) },
		func(cols []string) string { return fmt.Sprintf("請分析%s的趨勢", cols[0]) },
	}

	dualColTemplates := []templateFunc{
		func(cols []string) string { return fmt.Sprintf("%s和%s的關聯是什麼？", cols[0], cols[1]) },
		func(cols []string) string { return fmt.Sprintf("按%s分組，%s的平均值為何？", cols[0], cols[1]) },
		func(cols []string) string { return fmt.Sprintf("%s最高的項目，其%s是多少？", cols[0], cols[1]) },
	}

	// Shuffle headers to get variety
	shuffled := make([]string, len(headers))
	copy(shuffled, headers)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Generate first suggestion: dual-column if possible
	if len(shuffled) >= 2 {
		tmpl := dualColTemplates[rand.Intn(len(dualColTemplates))]
		suggestions = append(suggestions, tmpl(shuffled[:2]))
	} else {
		tmpl := singleColTemplates[rand.Intn(len(singleColTemplates))]
		suggestions = append(suggestions, tmpl(shuffled[:1]))
	}

	// Generate second suggestion: single-column
	colIdx := 0
	if len(shuffled) > 1 {
		colIdx = 1
	}
	tmpl := singleColTemplates[rand.Intn(len(singleColTemplates))]
	suggestions = append(suggestions, tmpl(shuffled[colIdx:colIdx+1]))

	// Generate third suggestion: use another column or dual-column
	if len(shuffled) >= 3 {
		tmpl2 := dualColTemplates[rand.Intn(len(dualColTemplates))]
		suggestions = append(suggestions, tmpl2(shuffled[1:3]))
	} else if len(shuffled) >= 2 {
		tmpl2 := singleColTemplates[rand.Intn(len(singleColTemplates))]
		suggestions = append(suggestions, tmpl2(shuffled[len(shuffled)-1:]))
	} else {
		suggestions = append(suggestions, fmt.Sprintf("請總結%s的整體情況", shuffled[0]))
	}

	return suggestions
}
